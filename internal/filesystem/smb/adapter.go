package smb

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	pathpkg "path"
	"strconv"
	"strings"
	"time"

	smb2 "github.com/hirochachacha/go-smb2"

	"voyager/internal/domain"
	"voyager/internal/filesystem/transferio"
)

type Adapter struct {
	address   string
	host      string
	shareName string
	username  string
	password  string
	domain    string
	rootPath  string
}

func NewAdapter(input domain.ConnectionInput) *Adapter {
	port := input.Port
	if port == 0 {
		port = 445
	}
	host := strings.TrimSpace(input.Host)
	return &Adapter{
		address:   net.JoinHostPort(host, strconv.Itoa(port)),
		host:      host,
		shareName: strings.Trim(strings.TrimSpace(input.Share), `/\`),
		username:  strings.TrimSpace(input.Username),
		password:  input.Password,
		domain:    strings.TrimSpace(input.Domain),
		rootPath:  cleanSharePath(input.RootPath),
	}
}

func (a *Adapter) Test(ctx context.Context) error {
	share, cleanup, err := a.connectShare(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	if a.rootPath == "" {
		return nil
	}
	_, err = share.Stat(a.rootPath)
	return err
}

func (a *Adapter) List(ctx context.Context, path string) ([]domain.RemoteEntry, error) {
	share, cleanup, err := a.connectShare(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	infos, err := share.ReadDir(a.sharePath(path))
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
			Name:       info.Name(),
			Path:       entryPath(path, info.Name()),
			Type:       entryType,
			Size:       info.Size(),
			ModifiedAt: modifiedAt,
		})
	}
	return entries, nil
}

func (a *Adapter) Mkdir(ctx context.Context, path string) error {
	share, cleanup, err := a.connectShare(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	return share.Mkdir(a.sharePath(path), 0o755)
}

func (a *Adapter) Rename(ctx context.Context, oldPath string, newPath string) error {
	share, cleanup, err := a.connectShare(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	return share.Rename(a.sharePath(oldPath), a.sharePath(newPath))
}

func (a *Adapter) Delete(ctx context.Context, path string) error {
	share, cleanup, err := a.connectShare(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	sharePath := a.sharePath(path)
	if err := share.Remove(sharePath); err == nil {
		return nil
	}
	return share.RemoveAll(sharePath)
}

func (a *Adapter) Upload(ctx context.Context, localPath string, remotePath string, progress func(bytesDone int64)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	localFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	share, cleanup, err := a.connectShare(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	remoteFile, err := share.Create(a.sharePath(remotePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(remoteFile, transferio.NewProgressReader(ctx, localFile, progress)); err != nil {
		_ = remoteFile.Close()
		return err
	}
	return remoteFile.Close()
}

func (a *Adapter) Download(ctx context.Context, remotePath string, localPath string, progress func(bytesDone int64)) error {
	share, cleanup, err := a.connectShare(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	remoteFile, err := share.Open(a.sharePath(remotePath))
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	tmpPath := localPath + ".part"
	localFile, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(localFile, transferio.NewProgressReader(ctx, remoteFile, progress)); err != nil {
		_ = localFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := localFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, localPath)
}

func (a *Adapter) connectShare(ctx context.Context) (*smb2.Share, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	dialer := net.Dialer{Timeout: 30 * time.Second}
	tcpConn, err := dialer.DialContext(ctx, "tcp", a.address)
	if err != nil {
		return nil, nil, err
	}

	session, err := (&smb2.Dialer{Initiator: &smb2.NTLMInitiator{
		User:     a.username,
		Password: a.password,
		Domain:   a.domain,
	}}).DialContext(ctx, tcpConn)
	if err != nil {
		_ = tcpConn.Close()
		return nil, nil, err
	}

	share, err := session.WithContext(ctx).Mount(fmt.Sprintf(`\\%s\%s`, a.host, a.shareName))
	if err != nil {
		_ = session.Logoff()
		_ = tcpConn.Close()
		return nil, nil, err
	}
	cleanup := func() {
		_ = share.Umount()
		_ = session.Logoff()
		_ = tcpConn.Close()
	}
	return share.WithContext(ctx), cleanup, nil
}

func (a *Adapter) sharePath(path string) string {
	cleaned := cleanSharePath(path)
	if a.rootPath == "" {
		return cleaned
	}
	if cleaned == "" {
		return a.rootPath
	}
	return pathpkg.Join(a.rootPath, cleaned)
}

func cleanSharePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" || path == `\` {
		return ""
	}
	path = strings.ReplaceAll(path, `\`, "/")
	return strings.Trim(pathpkg.Clean(path), "/")
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
