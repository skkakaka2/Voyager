package webdav

import (
	"context"
	"os"
	pathpkg "path"
	"strings"
	"time"

	"github.com/studio-b12/gowebdav"

	"voyager/internal/domain"
	"voyager/internal/filesystem/transferio"
)

type Adapter struct {
	client   *gowebdav.Client
	rootPath string
}

func NewAdapter(input domain.ConnectionInput) *Adapter {
	client := gowebdav.NewClient(input.BaseURL, input.Username, input.Password)
	client.SetTimeout(30 * time.Second)
	return &Adapter{
		client:   client,
		rootPath: cleanRemotePath(input.RootPath),
	}
}

func (a *Adapter) Test(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.rootPath == "/" {
		return a.client.Connect()
	}
	_, err := a.client.Stat(a.rootPath)
	return err
}

func (a *Adapter) List(ctx context.Context, path string) ([]domain.RemoteEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	infos, err := a.client.ReadDir(a.remotePath(path))
	if err != nil {
		return nil, err
	}

	entries := make([]domain.RemoteEntry, 0, len(infos))
	for _, info := range infos {
		entryType := domain.EntryFile
		if info.IsDir() {
			entryType = domain.EntryDirectory
		}
		modifiedAt := ""
		if !info.ModTime().IsZero() {
			modifiedAt = info.ModTime().UTC().Format(time.RFC3339)
		}
		entries = append(entries, domain.RemoteEntry{
			Name:       strings.TrimSuffix(info.Name(), "/"),
			Path:       entryPath(path, info.Name()),
			Type:       entryType,
			Size:       info.Size(),
			ModifiedAt: modifiedAt,
		})
	}
	return entries, nil
}

func (a *Adapter) Mkdir(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return a.client.Mkdir(a.remotePath(path), 0o755)
}

func (a *Adapter) Rename(ctx context.Context, oldPath string, newPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return a.client.Rename(a.remotePath(oldPath), a.remotePath(newPath), false)
}

func (a *Adapter) Delete(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return a.client.Remove(a.remotePath(path))
}

func (a *Adapter) Upload(ctx context.Context, localPath string, remotePath string, progress func(bytesDone int64)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return a.client.WriteStream(a.remotePath(remotePath), transferio.NewProgressReader(ctx, file, progress), 0o644)
}

func (a *Adapter) Download(ctx context.Context, remotePath string, localPath string, progress func(bytesDone int64)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	reader, err := a.client.ReadStream(a.remotePath(remotePath))
	if err != nil {
		return err
	}
	defer reader.Close()

	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.ReadFrom(transferio.NewProgressReader(ctx, reader, progress))
	return err
}

func (a *Adapter) remotePath(path string) string {
	cleaned := cleanRemotePath(path)
	if a.rootPath == "/" {
		return cleaned
	}
	if cleaned == "/" {
		return a.rootPath
	}
	return pathpkg.Join(a.rootPath, cleaned)
}

func cleanRemotePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "/"
	}
	return "/" + strings.Trim(pathpkg.Clean(path), "/")
}

func entryPath(parent string, name string) string {
	return cleanRemotePath(pathpkg.Join(cleanRemotePath(parent), strings.TrimSuffix(name, "/")))
}
