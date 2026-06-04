package ftp

import (
	"context"
	"io"
	"net"
	"os"
	pathpkg "path"
	"strconv"
	"strings"
	"time"

	ftpclient "github.com/jlaffaye/ftp"

	"voyager/internal/domain"
	"voyager/internal/filesystem/transferio"
)

type Adapter struct {
	address  string
	username string
	password string
	rootPath string
}

func NewAdapter(input domain.ConnectionInput) *Adapter {
	port := input.Port
	if port == 0 {
		port = 21
	}
	return &Adapter{
		address:  net.JoinHostPort(strings.TrimSpace(input.Host), strconv.Itoa(port)),
		username: strings.TrimSpace(input.Username),
		password: input.Password,
		rootPath: cleanRemotePath(input.RootPath),
	}
}

func (a *Adapter) Test(ctx context.Context) error {
	conn, err := a.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Quit()

	if a.rootPath != "/" {
		return conn.ChangeDir(a.rootPath)
	}
	return conn.NoOp()
}

func (a *Adapter) List(ctx context.Context, path string) ([]domain.RemoteEntry, error) {
	conn, err := a.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Quit()

	entries, err := conn.List(a.remotePath(path))
	if err != nil {
		return nil, err
	}

	result := make([]domain.RemoteEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Name == "." || entry.Name == ".." {
			continue
		}
		entryType := domain.EntryFile
		if entry.Type == ftpclient.EntryTypeFolder {
			entryType = domain.EntryDirectory
		}
		modifiedAt := ""
		if !entry.Time.IsZero() {
			modifiedAt = entry.Time.UTC().Format(time.RFC3339)
		}
		result = append(result, domain.RemoteEntry{
			Name:       strings.TrimSuffix(entry.Name, "/"),
			Path:       entryPath(path, entry.Name),
			Type:       entryType,
			Size:       int64(entry.Size),
			ModifiedAt: modifiedAt,
		})
	}
	return result, nil
}

func (a *Adapter) Mkdir(ctx context.Context, path string) error {
	conn, err := a.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Quit()
	return conn.MakeDir(a.remotePath(path))
}

func (a *Adapter) Rename(ctx context.Context, oldPath string, newPath string) error {
	conn, err := a.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Quit()
	return conn.Rename(a.remotePath(oldPath), a.remotePath(newPath))
}

func (a *Adapter) Delete(ctx context.Context, path string) error {
	conn, err := a.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Quit()

	remotePath := a.remotePath(path)
	if err := conn.Delete(remotePath); err == nil {
		return nil
	}
	return conn.RemoveDirRecur(remotePath)
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

	conn, err := a.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Quit()
	return conn.Stor(a.remotePath(remotePath), transferio.NewProgressReader(ctx, file, progress))
}

func (a *Adapter) Download(ctx context.Context, remotePath string, localPath string, progress func(bytesDone int64)) error {
	conn, err := a.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Quit()

	response, err := conn.Retr(a.remotePath(remotePath))
	if err != nil {
		return err
	}

	tmpPath := localPath + ".part"
	file, err := os.Create(tmpPath)
	if err != nil {
		_ = response.Close()
		return err
	}
	if _, err := io.Copy(file, transferio.NewProgressReader(ctx, response, progress)); err != nil {
		_ = response.Close()
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := response.Close(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, localPath)
}

func (a *Adapter) connect(ctx context.Context) (*ftpclient.ServerConn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conn, err := ftpclient.Dial(a.address, ftpclient.DialWithContext(ctx), ftpclient.DialWithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}
	if err := conn.Login(a.username, a.password); err != nil {
		_ = conn.Quit()
		return nil, err
	}
	return conn, nil
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
