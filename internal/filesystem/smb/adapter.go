package smb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	pathpkg "path"
	"sort"
	"strconv"
	"strings"
	"time"

	smb2 "github.com/hirochachacha/go-smb2"

	"voyager/internal/domain"
	"voyager/internal/filesystem/transferio"
)

var errShareRequired = errors.New("请先进入一个 SMB 共享")

type Adapter struct {
	address        string
	host           string
	shareName      string
	username       string
	password       string
	domain         string
	rootPath       string
	listShareNames func(context.Context) ([]string, error)
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
	if a.shareName == "" {
		_, cleanup, err := a.connectSession(ctx)
		if err != nil {
			return err
		}
		defer cleanup()
		return nil
	}

	share, cleanup, err := a.connectNamedShare(ctx, a.shareName)
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
	if a.shareName == "" && cleanRemotePath(path) == "/" {
		names, err := a.serverShareNames(ctx)
		if err != nil {
			return nil, err
		}
		sort.Strings(names)
		entries := make([]domain.RemoteEntry, 0, len(names))
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" || strings.HasSuffix(name, "$") {
				continue
			}
			entries = append(entries, domain.RemoteEntry{
				Name: name,
				Path: entryPath("/", name),
				Type: domain.EntryDirectory,
			})
		}
		return entries, nil
	}

	shareName, sharePath, err := a.resolveSharePath(path)
	if err != nil {
		return nil, err
	}
	share, cleanup, err := a.connectNamedShare(ctx, shareName)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	infos, err := share.ReadDir(sharePath)
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
	shareName, sharePath, err := a.resolveWritableSharePath(path)
	if err != nil {
		return err
	}
	share, cleanup, err := a.connectNamedShare(ctx, shareName)
	if err != nil {
		return err
	}
	defer cleanup()
	return share.Mkdir(sharePath, 0o755)
}

func (a *Adapter) Rename(ctx context.Context, oldPath string, newPath string) error {
	oldShareName, oldSharePath, err := a.resolveWritableSharePath(oldPath)
	if err != nil {
		return err
	}
	newShareName, newSharePath, err := a.resolveWritableSharePath(newPath)
	if err != nil {
		return err
	}
	if oldShareName != newShareName {
		return errors.New("不能跨 SMB 共享重命名")
	}
	share, cleanup, err := a.connectNamedShare(ctx, oldShareName)
	if err != nil {
		return err
	}
	defer cleanup()
	return share.Rename(oldSharePath, newSharePath)
}

func (a *Adapter) Delete(ctx context.Context, path string) error {
	shareName, sharePath, err := a.resolveWritableSharePath(path)
	if err != nil {
		return err
	}
	share, cleanup, err := a.connectNamedShare(ctx, shareName)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := share.Remove(sharePath); err == nil {
		return nil
	}
	return share.RemoveAll(sharePath)
}

func (a *Adapter) Upload(ctx context.Context, localPath string, remotePath string, progress func(bytesDone int64)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	shareName, sharePath, err := a.resolveWritableSharePath(remotePath)
	if err != nil {
		return err
	}
	localFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	share, cleanup, err := a.connectNamedShare(ctx, shareName)
	if err != nil {
		return err
	}
	defer cleanup()

	remoteFile, err := share.Create(sharePath)
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
	shareName, sharePath, err := a.resolveWritableSharePath(remotePath)
	if err != nil {
		return err
	}
	share, cleanup, err := a.connectNamedShare(ctx, shareName)
	if err != nil {
		return err
	}
	defer cleanup()

	remoteFile, err := share.Open(sharePath)
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

func (a *Adapter) connectSession(ctx context.Context) (*smb2.Session, func(), error) {
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

	cleanup := func() {
		_ = session.Logoff()
		_ = tcpConn.Close()
	}
	return session.WithContext(ctx), cleanup, nil
}

func (a *Adapter) connectNamedShare(ctx context.Context, shareName string) (*smb2.Share, func(), error) {
	if strings.TrimSpace(shareName) == "" {
		return nil, nil, errShareRequired
	}
	session, sessionCleanup, err := a.connectSession(ctx)
	if err != nil {
		return nil, nil, err
	}

	share, err := session.Mount(fmt.Sprintf(`\\%s\%s`, a.host, shareName))
	if err != nil {
		sessionCleanup()
		return nil, nil, err
	}
	cleanup := func() {
		_ = share.Umount()
		sessionCleanup()
	}
	return share.WithContext(ctx), cleanup, nil
}

func (a *Adapter) serverShareNames(ctx context.Context) ([]string, error) {
	if a.listShareNames != nil {
		return a.listShareNames(ctx)
	}
	session, cleanup, err := a.connectSession(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return session.ListSharenames()
}

func (a *Adapter) resolveSharePath(path string) (string, string, error) {
	if a.shareName != "" {
		return a.shareName, a.sharePath(path), nil
	}

	cleaned := cleanRemotePath(path)
	if cleaned == "/" {
		return "", "", errShareRequired
	}
	parts := strings.SplitN(strings.Trim(cleaned, "/"), "/", 2)
	shareName := strings.TrimSpace(parts[0])
	if shareName == "" {
		return "", "", errShareRequired
	}
	sharePath := ""
	if len(parts) > 1 {
		sharePath = cleanSharePath(parts[1])
	}
	if a.rootPath == "" {
		return shareName, sharePath, nil
	}
	if sharePath == "" {
		return shareName, a.rootPath, nil
	}
	return shareName, pathpkg.Join(a.rootPath, sharePath), nil
}

func (a *Adapter) resolveWritableSharePath(path string) (string, string, error) {
	shareName, sharePath, err := a.resolveSharePath(path)
	if err != nil {
		return "", "", err
	}
	if a.shareName == "" {
		cleaned := strings.Trim(cleanRemotePath(path), "/")
		if sharePath == "" || !strings.Contains(cleaned, "/") {
			return "", "", errShareRequired
		}
	}
	if shareName == "" {
		return "", "", errShareRequired
	}
	return shareName, sharePath, nil
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
