package appsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"voyager/internal/config"
	"voyager/internal/domain"
	"voyager/internal/filesystem"
)

type fakeCredentialStore struct {
	err     error
	getErr  error
	saved   map[string]string
	deleted []string
}

type fakeAdapterFactory struct {
	input   domain.ConnectionInput
	adapter filesystem.Adapter
	err     error
}

func (f *fakeAdapterFactory) New(input domain.ConnectionInput) (filesystem.Adapter, error) {
	f.input = input
	if f.err != nil {
		return nil, f.err
	}
	return f.adapter, nil
}

type fakeAdapter struct {
	tested           bool
	listed           string
	mkdir            string
	renamedOld       string
	renamedNew       string
	deleted          string
	uploadedLocal    string
	uploadedRemote   string
	downloadedRemote string
	downloadedLocal  string
	entries          []domain.RemoteEntry
	err              error
	uploadProgress   int64
	downloadProgress int64
	waitForCancel    bool
	uploadStarted    chan struct{}
	uploadRelease    chan struct{}
	downloadStarted  chan struct{}
	downloadRelease  chan struct{}
}

func (f *fakeAdapter) Test(context.Context) error {
	f.tested = true
	return f.err
}

func (f *fakeAdapter) List(_ context.Context, path string) ([]domain.RemoteEntry, error) {
	f.listed = path
	return f.entries, f.err
}

func (f *fakeAdapter) Mkdir(_ context.Context, path string) error {
	f.mkdir = path
	return f.err
}

func (f *fakeAdapter) Rename(_ context.Context, oldPath string, newPath string) error {
	f.renamedOld = oldPath
	f.renamedNew = newPath
	return f.err
}

func (f *fakeAdapter) Delete(_ context.Context, path string) error {
	f.deleted = path
	return f.err
}

func (f *fakeAdapter) Upload(ctx context.Context, localPath string, remotePath string, progress filesystem.ProgressFunc) error {
	f.uploadedLocal = localPath
	f.uploadedRemote = remotePath
	if f.uploadProgress > 0 && progress != nil {
		progress(f.uploadProgress)
	}
	if f.uploadStarted != nil {
		close(f.uploadStarted)
	}
	if f.waitForCancel {
		<-ctx.Done()
		return ctx.Err()
	}
	if f.uploadRelease != nil {
		<-f.uploadRelease
	}
	return f.err
}

func (f *fakeAdapter) Download(ctx context.Context, remotePath string, localPath string, progress filesystem.ProgressFunc) error {
	f.downloadedRemote = remotePath
	f.downloadedLocal = localPath
	if f.downloadProgress > 0 && progress != nil {
		progress(f.downloadProgress)
	}
	if f.downloadStarted != nil {
		close(f.downloadStarted)
	}
	if f.waitForCancel {
		<-ctx.Done()
		return ctx.Err()
	}
	if f.downloadRelease != nil {
		<-f.downloadRelease
	}
	return f.err
}

func (f *fakeCredentialStore) Save(_ context.Context, key string, password string) error {
	if f.err != nil {
		return f.err
	}
	if f.saved == nil {
		f.saved = map[string]string{}
	}
	f.saved[key] = password
	return nil
}

func (f *fakeCredentialStore) Get(_ context.Context, key string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	if f.saved == nil {
		return "", errors.New("missing credential")
	}
	password, ok := f.saved[key]
	if !ok {
		return "", errors.New("missing credential")
	}
	return password, nil
}

func (f *fakeCredentialStore) Delete(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	return nil
}

func TestSaveConnectionPersistsProfileWithoutPlainPassword(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	credentials := &fakeCredentialStore{}
	service := NewConnectionService(config.NewConnectionStore(storePath), credentials)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:         "Office WebDAV",
		Protocol:     domain.ProtocolWebDAV,
		BaseURL:      "https://files.example.com/dav",
		Username:     "ada",
		Password:     "super-secret",
		SavePassword: true,
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	if profile.ID == "" {
		t.Fatal("expected generated connection id")
	}
	if !profile.PasswordSaved {
		t.Fatal("expected password to be marked as saved")
	}
	if credentials.saved[profile.CredentialKey] != "super-secret" {
		t.Fatal("expected password to be stored in credential provider")
	}

	raw, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read store file: %v", err)
	}
	if strings.Contains(string(raw), "super-secret") || strings.Contains(strings.ToLower(string(raw)), `"password":`) {
		t.Fatalf("connection store leaked password data: %s", raw)
	}
}

func TestSaveConnectionFallsBackWhenCredentialStoreFails(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	adapter := &fakeAdapter{}
	factory := &fakeAdapterFactory{adapter: adapter}
	service := NewConnectionService(
		config.NewConnectionStore(storePath),
		&fakeCredentialStore{err: errors.New("keyring unavailable")},
		factory,
	)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:         "NAS FTP",
		Protocol:     domain.ProtocolFTP,
		Host:         "nas.local",
		Username:     "ada",
		Password:     "super-secret",
		SavePassword: true,
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}
	if profile.PasswordSaved {
		t.Fatal("expected password to be unsaved when credential store fails")
	}
	if profile.CredentialKey != "" {
		t.Fatalf("expected empty credential key, got %q", profile.CredentialKey)
	}

	if err := service.TestConnection(context.Background(), domain.ConnectionInput{ID: profile.ID}); err != nil {
		t.Fatalf("TestConnection returned error: %v", err)
	}
	if factory.input.Password != "super-secret" {
		t.Fatalf("expected session password to be passed to adapter, got %q", factory.input.Password)
	}
}

func TestDeleteConnectionRemovesProfileAndCredential(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	credentials := &fakeCredentialStore{}
	service := NewConnectionService(config.NewConnectionStore(storePath), credentials)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:         "Office WebDAV",
		Protocol:     domain.ProtocolWebDAV,
		BaseURL:      "https://files.example.com/dav",
		Username:     "ada",
		Password:     "super-secret",
		SavePassword: true,
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	if err := service.DeleteConnection(context.Background(), profile.ID); err != nil {
		t.Fatalf("DeleteConnection returned error: %v", err)
	}

	connections, err := service.ListConnections(context.Background())
	if err != nil {
		t.Fatalf("ListConnections returned error: %v", err)
	}
	if len(connections) != 0 {
		t.Fatalf("expected no connections, got %d", len(connections))
	}
	if len(credentials.deleted) != 1 || credentials.deleted[0] != profile.CredentialKey {
		t.Fatalf("expected credential %q to be deleted, got %#v", profile.CredentialKey, credentials.deleted)
	}
}

func TestGetConnectionPasswordReturnsStoredCredential(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	credentials := &fakeCredentialStore{}
	service := NewConnectionService(config.NewConnectionStore(storePath), credentials)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:         "Office WebDAV",
		Protocol:     domain.ProtocolWebDAV,
		BaseURL:      "https://files.example.com/dav",
		Username:     "ada",
		Password:     "super-secret",
		SavePassword: true,
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	password, err := service.GetConnectionPassword(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("GetConnectionPassword returned error: %v", err)
	}
	if password != "super-secret" {
		t.Fatalf("expected stored password, got %q", password)
	}
}

func TestGetConnectionPasswordReportsMissingCredential(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	service := NewConnectionService(config.NewConnectionStore(storePath), &fakeCredentialStore{})

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:     "Office WebDAV",
		Protocol: domain.ProtocolWebDAV,
		BaseURL:  "https://files.example.com/dav",
		Username: "ada",
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	_, err = service.GetConnectionPassword(context.Background(), profile.ID)
	if !errors.Is(err, ErrCredentialNotSaved) {
		t.Fatalf("expected ErrCredentialNotSaved, got %v", err)
	}
}

func TestTestConnectionUsesSavedPasswordForAdapter(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	credentials := &fakeCredentialStore{}
	adapter := &fakeAdapter{}
	factory := &fakeAdapterFactory{adapter: adapter}
	service := NewConnectionService(config.NewConnectionStore(storePath), credentials, factory)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:         "Office WebDAV",
		Protocol:     domain.ProtocolWebDAV,
		BaseURL:      "https://files.example.com/dav",
		Username:     "ada",
		Password:     "super-secret",
		SavePassword: true,
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	err = service.TestConnection(context.Background(), domain.ConnectionInput{
		ID:       profile.ID,
		Name:     profile.Name,
		Protocol: profile.Protocol,
		BaseURL:  profile.BaseURL,
		Username: profile.Username,
	})
	if err != nil {
		t.Fatalf("TestConnection returned error: %v", err)
	}

	if !adapter.tested {
		t.Fatal("expected adapter Test to be called")
	}
	if factory.input.Password != "super-secret" {
		t.Fatalf("expected saved password to be passed to adapter, got %q", factory.input.Password)
	}
}

func TestListFilesReturnsAdapterEntries(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	credentials := &fakeCredentialStore{}
	entries := []domain.RemoteEntry{
		{Name: "docs", Path: "/docs", Type: domain.EntryDirectory},
		{Name: "readme.txt", Path: "/readme.txt", Type: domain.EntryFile, Size: 42},
	}
	adapter := &fakeAdapter{entries: entries}
	service := NewConnectionService(
		config.NewConnectionStore(storePath),
		credentials,
		&fakeAdapterFactory{adapter: adapter},
	)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:         "Office WebDAV",
		Protocol:     domain.ProtocolWebDAV,
		BaseURL:      "https://files.example.com/dav",
		Username:     "ada",
		Password:     "super-secret",
		SavePassword: true,
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	got, err := service.ListFiles(context.Background(), profile.ID, "/")
	if err != nil {
		t.Fatalf("ListFiles returned error: %v", err)
	}
	if adapter.listed != "/" {
		t.Fatalf("expected adapter to list /, got %q", adapter.listed)
	}
	if len(got) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(got))
	}
}

func TestCreateFolderCallsAdapterWithChildPath(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	adapter := &fakeAdapter{}
	service := NewConnectionService(
		config.NewConnectionStore(storePath),
		&fakeCredentialStore{},
		&fakeAdapterFactory{adapter: adapter},
	)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:     "Office WebDAV",
		Protocol: domain.ProtocolWebDAV,
		BaseURL:  "https://files.example.com/dav",
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	if err := service.CreateFolder(context.Background(), profile.ID, "/", "Projects"); err != nil {
		t.Fatalf("CreateFolder returned error: %v", err)
	}
	if adapter.mkdir != "/Projects" {
		t.Fatalf("expected mkdir /Projects, got %q", adapter.mkdir)
	}
}

func TestRenameEntryKeepsParentPath(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	adapter := &fakeAdapter{}
	service := NewConnectionService(
		config.NewConnectionStore(storePath),
		&fakeCredentialStore{},
		&fakeAdapterFactory{adapter: adapter},
	)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:     "Office WebDAV",
		Protocol: domain.ProtocolWebDAV,
		BaseURL:  "https://files.example.com/dav",
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	if err := service.RenameEntry(context.Background(), profile.ID, "/docs/old.txt", "new.txt"); err != nil {
		t.Fatalf("RenameEntry returned error: %v", err)
	}
	if adapter.renamedOld != "/docs/old.txt" || adapter.renamedNew != "/docs/new.txt" {
		t.Fatalf("expected rename /docs/old.txt -> /docs/new.txt, got %q -> %q", adapter.renamedOld, adapter.renamedNew)
	}
}

func TestDeleteEntryCallsAdapter(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	adapter := &fakeAdapter{}
	service := NewConnectionService(
		config.NewConnectionStore(storePath),
		&fakeCredentialStore{},
		&fakeAdapterFactory{adapter: adapter},
	)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:     "Office WebDAV",
		Protocol: domain.ProtocolWebDAV,
		BaseURL:  "https://files.example.com/dav",
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	if err := service.DeleteEntry(context.Background(), profile.ID, "/docs"); err != nil {
		t.Fatalf("DeleteEntry returned error: %v", err)
	}
	if adapter.deleted != "/docs" {
		t.Fatalf("expected delete /docs, got %q", adapter.deleted)
	}
}

func TestUploadFilesRecordsSucceededTask(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	localFile := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(localFile, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}
	adapter := &fakeAdapter{}
	service := NewConnectionService(
		config.NewConnectionStore(storePath),
		&fakeCredentialStore{},
		&fakeAdapterFactory{adapter: adapter},
	)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:     "Office WebDAV",
		Protocol: domain.ProtocolWebDAV,
		BaseURL:  "https://files.example.com/dav",
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	tasks, err := service.UploadFiles(context.Background(), profile.ID, "/docs", []string{localFile})
	if err != nil {
		t.Fatalf("UploadFiles returned error: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Status != domain.TransferRunning {
		t.Fatalf("expected one running task, got %#v", tasks)
	}
	eventually(t, func() bool {
		listed, err := service.ListTransferTasks(context.Background())
		return err == nil &&
			len(listed) == 1 &&
			listed[0].Status == domain.TransferSucceeded &&
			listed[0].Destination == "/docs/report.txt" &&
			adapter.uploadedLocal == localFile &&
			adapter.uploadedRemote == "/docs/report.txt"
	})
}

func TestUploadFilesReturnsRunningTaskBeforeTransferCompletes(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	localFile := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(localFile, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}
	adapter := &fakeAdapter{
		uploadStarted: make(chan struct{}),
		uploadRelease: make(chan struct{}),
	}
	service := NewConnectionService(
		config.NewConnectionStore(storePath),
		&fakeCredentialStore{},
		&fakeAdapterFactory{adapter: adapter},
	)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:     "Office WebDAV",
		Protocol: domain.ProtocolWebDAV,
		BaseURL:  "https://files.example.com/dav",
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	type uploadResult struct {
		tasks []domain.TransferTask
		err   error
	}
	resultCh := make(chan uploadResult, 1)
	go func() {
		tasks, err := service.UploadFiles(context.Background(), profile.ID, "/docs", []string{localFile})
		resultCh <- uploadResult{tasks: tasks, err: err}
	}()

	var result uploadResult
	select {
	case result = <-resultCh:
	case <-time.After(100 * time.Millisecond):
		close(adapter.uploadRelease)
		t.Fatal("UploadFiles did not return before transfer completed")
	}
	if result.err != nil {
		t.Fatalf("UploadFiles returned error: %v", result.err)
	}
	if len(result.tasks) != 1 || result.tasks[0].Status != domain.TransferRunning {
		t.Fatalf("expected one running task, got %#v", result.tasks)
	}
	listed, err := service.ListTransferTasks(context.Background())
	if err != nil {
		t.Fatalf("ListTransferTasks returned error: %v", err)
	}
	if len(listed) != 1 || listed[0].Status != domain.TransferRunning {
		t.Fatalf("expected listed running task, got %#v", listed)
	}

	close(adapter.uploadRelease)
	eventually(t, func() bool {
		listed, err := service.ListTransferTasks(context.Background())
		return err == nil && len(listed) == 1 && listed[0].Status == domain.TransferSucceeded
	})
}

func TestUploadFilesUpdatesProgressBeforeTransferCompletes(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	localFile := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(localFile, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}
	adapter := &fakeAdapter{
		uploadProgress: 3,
		uploadStarted:  make(chan struct{}),
		uploadRelease:  make(chan struct{}),
	}
	service := NewConnectionService(
		config.NewConnectionStore(storePath),
		&fakeCredentialStore{},
		&fakeAdapterFactory{adapter: adapter},
	)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:     "Office WebDAV",
		Protocol: domain.ProtocolWebDAV,
		BaseURL:  "https://files.example.com/dav",
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	if _, err := service.UploadFiles(context.Background(), profile.ID, "/docs", []string{localFile}); err != nil {
		t.Fatalf("UploadFiles returned error: %v", err)
	}
	<-adapter.uploadStarted
	listed, err := service.ListTransferTasks(context.Background())
	if err != nil {
		t.Fatalf("ListTransferTasks returned error: %v", err)
	}
	if len(listed) != 1 || listed[0].Status != domain.TransferRunning || listed[0].BytesDone != 3 {
		t.Fatalf("expected running task with 3 bytes done, got %#v", listed)
	}

	close(adapter.uploadRelease)
	eventually(t, func() bool {
		listed, err := service.ListTransferTasks(context.Background())
		return err == nil && len(listed) == 1 && listed[0].Status == domain.TransferSucceeded
	})
}

func TestCancelTransferTaskMarksRunningTaskCanceled(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	localFile := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(localFile, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}
	adapter := &fakeAdapter{
		waitForCancel: true,
		uploadStarted: make(chan struct{}),
	}
	service := NewConnectionService(
		config.NewConnectionStore(storePath),
		&fakeCredentialStore{},
		&fakeAdapterFactory{adapter: adapter},
	)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:     "Office WebDAV",
		Protocol: domain.ProtocolWebDAV,
		BaseURL:  "https://files.example.com/dav",
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	tasks, err := service.UploadFiles(context.Background(), profile.ID, "/docs", []string{localFile})
	if err != nil {
		t.Fatalf("UploadFiles returned error: %v", err)
	}
	<-adapter.uploadStarted
	if err := service.CancelTransferTask(context.Background(), tasks[0].ID); err != nil {
		t.Fatalf("CancelTransferTask returned error: %v", err)
	}

	eventually(t, func() bool {
		listed, err := service.ListTransferTasks(context.Background())
		return err == nil && len(listed) == 1 && listed[0].Status == domain.TransferCanceled
	})
}

func TestDownloadFileRecordsSucceededTask(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "connections.json")
	localDir := t.TempDir()
	adapter := &fakeAdapter{}
	service := NewConnectionService(
		config.NewConnectionStore(storePath),
		&fakeCredentialStore{},
		&fakeAdapterFactory{adapter: adapter},
	)

	profile, err := service.SaveConnection(context.Background(), domain.ConnectionInput{
		Name:     "Office WebDAV",
		Protocol: domain.ProtocolWebDAV,
		BaseURL:  "https://files.example.com/dav",
	})
	if err != nil {
		t.Fatalf("SaveConnection returned error: %v", err)
	}

	task, err := service.DownloadFile(context.Background(), profile.ID, "/docs/report.txt", localDir, 42)
	if err != nil {
		t.Fatalf("DownloadFile returned error: %v", err)
	}
	expectedLocal := filepath.Join(localDir, "report.txt")
	if task.Status != domain.TransferRunning || task.Destination != expectedLocal || task.BytesTotal != 42 {
		t.Fatalf("expected running download task, got %#v", task)
	}
	eventually(t, func() bool {
		listed, err := service.ListTransferTasks(context.Background())
		return err == nil &&
			len(listed) == 1 &&
			listed[0].Status == domain.TransferSucceeded &&
			listed[0].Destination == expectedLocal &&
			adapter.downloadedRemote == "/docs/report.txt" &&
			adapter.downloadedLocal == expectedLocal
	})
}

func eventually(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
