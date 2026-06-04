package appsvc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"voyager/internal/domain"
	"voyager/internal/filesystem"
)

var (
	ErrConnectionNotFound     = errors.New("connection not found")
	ErrCredentialNotSaved     = errors.New("connection credential is not saved")
	ErrProtocolNotImplemented = errors.New("protocol adapter is not implemented yet")
)

type ConnectionRepository interface {
	Load() ([]domain.ConnectionProfile, error)
	Save(connections []domain.ConnectionProfile) error
}

type CredentialStore interface {
	Get(ctx context.Context, key string) (string, error)
	Save(ctx context.Context, key string, password string) error
	Delete(ctx context.Context, key string) error
}

type AdapterFactory interface {
	New(input domain.ConnectionInput) (filesystem.Adapter, error)
}

type ConnectionService struct {
	repository  ConnectionRepository
	credentials CredentialStore
	adapters    AdapterFactory
	tasksMu     sync.Mutex
	tasks       []domain.TransferTask
	cancelsMu   sync.Mutex
	cancels     map[string]context.CancelFunc
	passwordsMu sync.Mutex
	passwords   map[string]string
}

func NewConnectionService(repository ConnectionRepository, credentials CredentialStore, adapters ...AdapterFactory) *ConnectionService {
	var factory AdapterFactory
	if len(adapters) > 0 {
		factory = adapters[0]
	}
	return &ConnectionService{repository: repository, credentials: credentials, adapters: factory}
}

func (s *ConnectionService) ListConnections(context.Context) ([]domain.ConnectionProfile, error) {
	return s.repository.Load()
}

func (s *ConnectionService) SaveConnection(ctx context.Context, input domain.ConnectionInput) (domain.ConnectionProfile, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return domain.ConnectionProfile{}, errors.New("connection name is required")
	}
	if input.Protocol != domain.ProtocolSMB && input.Protocol != domain.ProtocolWebDAV && input.Protocol != domain.ProtocolFTP {
		return domain.ConnectionProfile{}, fmt.Errorf("unsupported protocol: %s", input.Protocol)
	}

	connections, err := s.repository.Load()
	if err != nil {
		return domain.ConnectionProfile{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	index := -1
	profile := domain.ConnectionProfile{CreatedAt: now}
	if input.ID != "" {
		for i, item := range connections {
			if item.ID == input.ID {
				index = i
				profile = item
				break
			}
		}
	}
	if profile.ID == "" {
		id, err := randomID()
		if err != nil {
			return domain.ConnectionProfile{}, err
		}
		profile.ID = id
	}

	profile.Name = input.Name
	profile.Protocol = input.Protocol
	profile.Host = strings.TrimSpace(input.Host)
	profile.Port = input.Port
	profile.BaseURL = strings.TrimSpace(input.BaseURL)
	profile.Share = strings.Trim(strings.TrimSpace(input.Share), "/")
	profile.RootPath = strings.TrimSpace(input.RootPath)
	profile.Username = strings.TrimSpace(input.Username)
	profile.Domain = strings.TrimSpace(input.Domain)
	profile.PassiveMode = input.PassiveMode
	profile.UpdatedAt = now
	if profile.Port == 0 {
		switch profile.Protocol {
		case domain.ProtocolSMB:
			profile.Port = 445
		case domain.ProtocolFTP:
			profile.Port = 21
		}
	}

	if input.Password != "" {
		profile.CredentialKey = ""
		profile.PasswordSaved = false
		if input.SavePassword {
			if err := s.credentials.Save(ctx, profile.ID, input.Password); err == nil {
				profile.CredentialKey = profile.ID
				profile.PasswordSaved = true
			}
		}
		s.rememberSessionPassword(profile.ID, input.Password)
	} else if !input.SavePassword && profile.CredentialKey != "" {
		_ = s.credentials.Delete(ctx, profile.CredentialKey)
		profile.CredentialKey = ""
		profile.PasswordSaved = false
	}

	if index >= 0 {
		connections[index] = profile
	} else {
		connections = append(connections, profile)
	}
	if err := s.repository.Save(connections); err != nil {
		return domain.ConnectionProfile{}, err
	}
	return profile, nil
}

func (s *ConnectionService) DeleteConnection(ctx context.Context, id string) error {
	connections, err := s.repository.Load()
	if err != nil {
		return err
	}

	next := connections[:0]
	var credentialKey string
	for _, item := range connections {
		if item.ID == id {
			credentialKey = item.CredentialKey
			continue
		}
		next = append(next, item)
	}
	if err := s.repository.Save(next); err != nil {
		return err
	}
	s.forgetSessionPassword(id)
	if credentialKey != "" {
		return s.credentials.Delete(ctx, credentialKey)
	}
	return nil
}

func (s *ConnectionService) GetConnectionPassword(ctx context.Context, id string) (string, error) {
	connections, err := s.repository.Load()
	if err != nil {
		return "", err
	}
	for _, item := range connections {
		if item.ID != id {
			continue
		}
		if item.CredentialKey == "" {
			return "", ErrCredentialNotSaved
		}
		return s.credentials.Get(ctx, item.CredentialKey)
	}
	return "", ErrConnectionNotFound
}

func (s *ConnectionService) TestConnection(ctx context.Context, input domain.ConnectionInput) error {
	adapter, err := s.adapterForInput(ctx, input)
	if err != nil {
		return localizeError(err)
	}
	return localizeError(adapter.Test(ctx))
}

func (s *ConnectionService) ListFiles(ctx context.Context, connectionID string, path string) ([]domain.RemoteEntry, error) {
	adapter, err := s.adapterForConnection(ctx, connectionID)
	if err != nil {
		return nil, localizeError(err)
	}
	entries, err := adapter.List(ctx, path)
	if err != nil {
		return nil, localizeError(err)
	}
	return entries, nil
}

func (s *ConnectionService) CreateFolder(ctx context.Context, connectionID string, parentPath string, name string) error {
	adapter, err := s.adapterForConnection(ctx, connectionID)
	if err != nil {
		return localizeError(err)
	}
	return localizeError(adapter.Mkdir(ctx, childRemotePath(parentPath, name)))
}

func (s *ConnectionService) RenameEntry(ctx context.Context, connectionID string, oldPath string, newName string) error {
	adapter, err := s.adapterForConnection(ctx, connectionID)
	if err != nil {
		return localizeError(err)
	}
	oldPath = cleanServicePath(oldPath)
	return localizeError(adapter.Rename(ctx, oldPath, childRemotePath(pathpkg.Dir(oldPath), newName)))
}

func (s *ConnectionService) DeleteEntry(ctx context.Context, connectionID string, path string) error {
	adapter, err := s.adapterForConnection(ctx, connectionID)
	if err != nil {
		return localizeError(err)
	}
	return localizeError(adapter.Delete(ctx, cleanServicePath(path)))
}

func (s *ConnectionService) UploadFiles(ctx context.Context, connectionID string, remoteDir string, localPaths []string) ([]domain.TransferTask, error) {
	if _, err := s.adapterForConnection(ctx, connectionID); err != nil {
		return nil, localizeError(err)
	}

	tasks := make([]domain.TransferTask, 0, len(localPaths))
	for _, localPath := range localPaths {
		remotePath := childRemotePath(remoteDir, filepath.Base(localPath))
		task := newTransferTask(connectionID, domain.TransferUpload, localPath, remotePath)
		if info, err := os.Stat(localPath); err == nil {
			task.BytesTotal = info.Size()
		}

		startedAt := time.Now().UTC().Format(time.RFC3339)
		task.StartedAt = &startedAt
		task.Status = domain.TransferRunning
		s.recordTransferTask(task)
		tasks = append(tasks, task)
		s.startTransferTask(task, func(runCtx context.Context, progress filesystem.ProgressFunc) (int64, error) {
			adapter, err := s.adapterForConnection(runCtx, connectionID)
			if err != nil {
				return 0, localizeError(err)
			}
			return task.BytesTotal, localizeError(adapter.Upload(runCtx, task.Source, task.Destination, progress))
		})
	}

	return tasks, nil
}

func (s *ConnectionService) DownloadFile(ctx context.Context, connectionID string, remotePath string, localDir string, bytesTotal int64) (domain.TransferTask, error) {
	if _, err := s.adapterForConnection(ctx, connectionID); err != nil {
		return domain.TransferTask{}, localizeError(err)
	}

	cleanedRemotePath := cleanServicePath(remotePath)
	localPath := filepath.Join(localDir, pathpkg.Base(cleanedRemotePath))
	task := newTransferTask(connectionID, domain.TransferDownload, cleanedRemotePath, localPath)
	task.BytesTotal = bytesTotal
	startedAt := time.Now().UTC().Format(time.RFC3339)
	task.StartedAt = &startedAt
	task.Status = domain.TransferRunning
	s.recordTransferTask(task)
	s.startTransferTask(task, func(runCtx context.Context, progress filesystem.ProgressFunc) (int64, error) {
		adapter, err := s.adapterForConnection(runCtx, connectionID)
		if err != nil {
			return 0, localizeError(err)
		}
		if err := localizeError(adapter.Download(runCtx, task.Source, task.Destination, progress)); err != nil {
			return 0, err
		}
		if bytesTotal > 0 {
			return bytesTotal, nil
		}
		if info, statErr := os.Stat(task.Destination); statErr == nil {
			return info.Size(), nil
		}
		return 0, nil
	})

	return task, nil
}

func (s *ConnectionService) ListTransferTasks(context.Context) ([]domain.TransferTask, error) {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	tasks := make([]domain.TransferTask, len(s.tasks))
	copy(tasks, s.tasks)
	return tasks, nil
}

func (s *ConnectionService) CancelTransferTask(_ context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("transfer task id is required")
	}

	s.cancelsMu.Lock()
	cancel := s.cancels[id]
	s.cancelsMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if !s.markTransferCanceled(id) {
		return errors.New("transfer task not found")
	}
	return nil
}

func randomID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func (s *ConnectionService) inputForConnection(ctx context.Context, id string) (domain.ConnectionInput, error) {
	connections, err := s.repository.Load()
	if err != nil {
		return domain.ConnectionInput{}, err
	}
	for _, item := range connections {
		if item.ID != id {
			continue
		}
		input := domain.ConnectionInput{
			ID:          item.ID,
			Name:        item.Name,
			Protocol:    item.Protocol,
			Host:        item.Host,
			Port:        item.Port,
			BaseURL:     item.BaseURL,
			Share:       item.Share,
			RootPath:    item.RootPath,
			Username:    item.Username,
			Domain:      item.Domain,
			PassiveMode: item.PassiveMode,
		}
		if item.CredentialKey != "" {
			password, err := s.credentials.Get(ctx, item.CredentialKey)
			if err == nil {
				input.Password = password
			} else if password, ok := s.sessionPassword(item.ID); ok {
				input.Password = password
			} else {
				return domain.ConnectionInput{}, err
			}
		} else if password, ok := s.sessionPassword(item.ID); ok {
			input.Password = password
		}
		return input, nil
	}
	return domain.ConnectionInput{}, ErrConnectionNotFound
}

func (s *ConnectionService) adapterForInput(ctx context.Context, input domain.ConnectionInput) (filesystem.Adapter, error) {
	if s.adapters == nil {
		return nil, ErrProtocolNotImplemented
	}
	if input.ID != "" && input.Password == "" {
		saved, err := s.inputForConnection(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		if input.Name == "" {
			input.Name = saved.Name
		}
		if input.Protocol == "" {
			input.Protocol = saved.Protocol
		}
		if input.Host == "" {
			input.Host = saved.Host
		}
		if input.Port == 0 {
			input.Port = saved.Port
		}
		if input.BaseURL == "" {
			input.BaseURL = saved.BaseURL
		}
		if input.Share == "" {
			input.Share = saved.Share
		}
		if input.RootPath == "" {
			input.RootPath = saved.RootPath
		}
		if input.Username == "" {
			input.Username = saved.Username
		}
		if input.Domain == "" {
			input.Domain = saved.Domain
		}
		input.PassiveMode = input.PassiveMode || saved.PassiveMode
		input.Password = saved.Password
	}
	return s.adapters.New(input)
}

func (s *ConnectionService) adapterForConnection(ctx context.Context, connectionID string) (filesystem.Adapter, error) {
	input, err := s.inputForConnection(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return s.adapterForInput(ctx, input)
}

func cleanServicePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "/"
	}
	return "/" + strings.Trim(pathpkg.Clean(path), "/")
}

func childRemotePath(parentPath string, name string) string {
	return cleanServicePath(pathpkg.Join(cleanServicePath(parentPath), strings.Trim(name, "/")))
}

func newTransferTask(connectionID string, direction domain.TransferDirection, source string, destination string) domain.TransferTask {
	now := time.Now().UTC().Format(time.RFC3339)
	id, err := randomID()
	if err != nil {
		id = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return domain.TransferTask{
		ID:           id,
		ConnectionID: connectionID,
		Direction:    direction,
		Source:       source,
		Destination:  destination,
		Status:       domain.TransferQueued,
		CreatedAt:    now,
	}
}

func (s *ConnectionService) recordTransferTask(task domain.TransferTask) {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	s.tasks = append([]domain.TransferTask{task}, s.tasks...)
}

func (s *ConnectionService) updateTransferTask(task domain.TransferTask) {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	for i := range s.tasks {
		if s.tasks[i].ID == task.ID {
			s.tasks[i] = task
			return
		}
	}
	s.tasks = append([]domain.TransferTask{task}, s.tasks...)
}

func (s *ConnectionService) startTransferTask(task domain.TransferTask, run func(context.Context, filesystem.ProgressFunc) (int64, error)) {
	runCtx, cancel := context.WithCancel(context.Background())
	s.rememberTransferCancel(task.ID, cancel)
	go func() {
		defer s.forgetTransferCancel(task.ID)

		bytesTotal, err := run(runCtx, func(bytesDone int64) {
			s.updateTransferProgress(task.ID, bytesDone)
		})

		status := domain.TransferSucceeded
		errorMessage := ""
		if err != nil {
			if errors.Is(err, context.Canceled) {
				status = domain.TransferCanceled
			} else {
				status = domain.TransferFailed
				errorMessage = localizeErrorMessage(err.Error())
			}
		}
		s.finishTransferTask(task.ID, status, bytesTotal, errorMessage)
	}()
}

func (s *ConnectionService) updateTransferProgress(id string, bytesDone int64) {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	for i := range s.tasks {
		if s.tasks[i].ID != id || s.tasks[i].Status != domain.TransferRunning {
			continue
		}
		if bytesDone > s.tasks[i].BytesDone {
			s.tasks[i].BytesDone = bytesDone
		}
		return
	}
}

func (s *ConnectionService) finishTransferTask(id string, status domain.TransferStatus, bytesTotal int64, errorMessage string) {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	finishedAt := time.Now().UTC().Format(time.RFC3339)
	for i := range s.tasks {
		if s.tasks[i].ID != id {
			continue
		}
		if s.tasks[i].Status == domain.TransferCanceled && status != domain.TransferCanceled {
			return
		}
		s.tasks[i].Status = status
		s.tasks[i].FinishedAt = &finishedAt
		s.tasks[i].ErrorMessage = errorMessage
		if bytesTotal > 0 {
			s.tasks[i].BytesTotal = bytesTotal
		}
		if status == domain.TransferSucceeded && s.tasks[i].BytesTotal > 0 {
			s.tasks[i].BytesDone = s.tasks[i].BytesTotal
		}
		return
	}
}

func (s *ConnectionService) markTransferCanceled(id string) bool {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()
	finishedAt := time.Now().UTC().Format(time.RFC3339)
	for i := range s.tasks {
		if s.tasks[i].ID != id {
			continue
		}
		if s.tasks[i].Status == domain.TransferQueued || s.tasks[i].Status == domain.TransferRunning {
			s.tasks[i].Status = domain.TransferCanceled
			s.tasks[i].FinishedAt = &finishedAt
			s.tasks[i].ErrorMessage = ""
		}
		return true
	}
	return false
}

func (s *ConnectionService) rememberTransferCancel(id string, cancel context.CancelFunc) {
	s.cancelsMu.Lock()
	defer s.cancelsMu.Unlock()
	if s.cancels == nil {
		s.cancels = map[string]context.CancelFunc{}
	}
	s.cancels[id] = cancel
}

func (s *ConnectionService) forgetTransferCancel(id string) {
	s.cancelsMu.Lock()
	defer s.cancelsMu.Unlock()
	delete(s.cancels, id)
}

func (s *ConnectionService) rememberSessionPassword(id string, password string) {
	s.passwordsMu.Lock()
	defer s.passwordsMu.Unlock()
	if s.passwords == nil {
		s.passwords = map[string]string{}
	}
	s.passwords[id] = password
}

func (s *ConnectionService) sessionPassword(id string) (string, bool) {
	s.passwordsMu.Lock()
	defer s.passwordsMu.Unlock()
	password, ok := s.passwords[id]
	return password, ok
}

func (s *ConnectionService) forgetSessionPassword(id string) {
	s.passwordsMu.Lock()
	defer s.passwordsMu.Unlock()
	delete(s.passwords, id)
}
