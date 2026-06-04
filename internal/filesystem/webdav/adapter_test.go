package webdav

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"net/http/httptest"

	"golang.org/x/net/webdav"

	"voyager/internal/domain"
)

func TestAdapterListsDirectoryEntries(t *testing.T) {
	ctx := context.Background()
	fs := webdav.NewMemFS()
	if err := fs.Mkdir(ctx, "/docs", 0o755); err != nil {
		t.Fatalf("create webdav dir: %v", err)
	}
	file, err := fs.OpenFile(ctx, "/readme.txt", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("create webdav file: %v", err)
	}
	if _, err := file.Write([]byte("hello")); err != nil {
		t.Fatalf("write webdav file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close webdav file: %v", err)
	}

	server := httptest.NewServer(&webdav.Handler{
		FileSystem: fs,
		LockSystem: webdav.NewMemLS(),
	})
	defer server.Close()

	adapter := NewAdapter(domain.ConnectionInput{
		Protocol: domain.ProtocolWebDAV,
		BaseURL:  server.URL,
	})
	entries, err := adapter.List(ctx, "/")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	got := map[string]domain.RemoteEntry{}
	for _, entry := range entries {
		got[entry.Path] = entry
	}
	if got["/docs"].Type != domain.EntryDirectory {
		t.Fatalf("expected /docs directory entry, got %#v", got["/docs"])
	}
	if got["/readme.txt"].Type != domain.EntryFile {
		t.Fatalf("expected /readme.txt file entry, got %#v", got["/readme.txt"])
	}
	if got["/readme.txt"].Size != 5 {
		t.Fatalf("expected /readme.txt size 5, got %d", got["/readme.txt"].Size)
	}
}

func TestAdapterCreatesRenamesAndDeletesEntry(t *testing.T) {
	ctx := context.Background()
	fs := webdav.NewMemFS()
	server := httptest.NewServer(&webdav.Handler{
		FileSystem: fs,
		LockSystem: webdav.NewMemLS(),
	})
	defer server.Close()

	adapter := NewAdapter(domain.ConnectionInput{
		Protocol: domain.ProtocolWebDAV,
		BaseURL:  server.URL,
	})

	if err := adapter.Mkdir(ctx, "/docs"); err != nil {
		t.Fatalf("Mkdir returned error: %v", err)
	}
	if _, err := fs.Stat(ctx, "/docs"); err != nil {
		t.Fatalf("expected /docs to exist: %v", err)
	}

	if err := adapter.Rename(ctx, "/docs", "/archive"); err != nil {
		t.Fatalf("Rename returned error: %v", err)
	}
	if _, err := fs.Stat(ctx, "/archive"); err != nil {
		t.Fatalf("expected /archive to exist: %v", err)
	}

	if err := adapter.Delete(ctx, "/archive"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := fs.Stat(ctx, "/archive"); err == nil {
		t.Fatal("expected /archive to be deleted")
	}
}

func TestAdapterUploadsAndDownloadsFile(t *testing.T) {
	ctx := context.Background()
	fs := webdav.NewMemFS()
	server := httptest.NewServer(&webdav.Handler{
		FileSystem: fs,
		LockSystem: webdav.NewMemLS(),
	})
	defer server.Close()

	adapter := NewAdapter(domain.ConnectionInput{
		Protocol: domain.ProtocolWebDAV,
		BaseURL:  server.URL,
	})

	localUpload := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(localUpload, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write local upload: %v", err)
	}
	var uploadProgress int64
	if err := adapter.Upload(ctx, localUpload, "/upload.txt", func(bytesDone int64) {
		uploadProgress = bytesDone
	}); err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}
	if uploadProgress != 5 {
		t.Fatalf("expected upload progress 5, got %d", uploadProgress)
	}
	uploaded, err := fs.OpenFile(ctx, "/upload.txt", os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("open uploaded webdav file: %v", err)
	}
	uploaded.Close()

	localDownload := filepath.Join(t.TempDir(), "download.txt")
	var downloadProgress int64
	if err := adapter.Download(ctx, "/upload.txt", localDownload, func(bytesDone int64) {
		downloadProgress = bytesDone
	}); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	if downloadProgress != 5 {
		t.Fatalf("expected download progress 5, got %d", downloadProgress)
	}
	content, err := os.ReadFile(localDownload)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("expected downloaded content hello, got %q", content)
	}
}
