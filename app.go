package main

import (
	"context"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"voyager/internal/appsvc"
	"voyager/internal/config"
	"voyager/internal/credential"
	"voyager/internal/domain"
	"voyager/internal/filesystem"
)

// App struct
type App struct {
	ctx         context.Context
	connections *appsvc.ConnectionService
}

// NewApp creates a new App application struct
func NewApp() *App {
	store, err := config.NewDefaultConnectionStore()
	if err != nil {
		store = config.NewConnectionStore(filepath.Join(".", "connections.json"))
	}
	return &App{
		connections: appsvc.NewConnectionService(
			store,
			credential.NewKeyringStore(credential.ServiceName),
			filesystem.NewFactory(),
		),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) ListConnections() ([]domain.ConnectionProfile, error) {
	return a.connections.ListConnections(a.callContext())
}

func (a *App) SaveConnection(input domain.ConnectionInput) (domain.ConnectionProfile, error) {
	return a.connections.SaveConnection(a.callContext(), input)
}

func (a *App) DeleteConnection(id string) error {
	return a.connections.DeleteConnection(a.callContext(), id)
}

func (a *App) TestConnection(input domain.ConnectionInput) error {
	return a.connections.TestConnection(a.callContext(), input)
}

func (a *App) ListFiles(connectionID string, path string) ([]domain.RemoteEntry, error) {
	return a.connections.ListFiles(a.callContext(), connectionID, path)
}

func (a *App) CreateFolder(connectionID string, parentPath string, name string) error {
	return a.connections.CreateFolder(a.callContext(), connectionID, parentPath, name)
}

func (a *App) RenameEntry(connectionID string, oldPath string, newName string) error {
	return a.connections.RenameEntry(a.callContext(), connectionID, oldPath, newName)
}

func (a *App) DeleteEntry(connectionID string, path string) error {
	return a.connections.DeleteEntry(a.callContext(), connectionID, path)
}

func (a *App) PickUploadFiles() ([]string, error) {
	runtime.LogInfo(a.callContext(), "PickUploadFiles called")
	return runtime.OpenMultipleFilesDialog(a.callContext(), runtime.OpenDialogOptions{
		Title: "选择要上传的文件",
	})
}

func (a *App) UploadFiles(connectionID string, remoteDir string, localPaths []string) ([]domain.TransferTask, error) {
	runtime.LogInfof(a.callContext(), "UploadFiles called: connection=%s remoteDir=%s files=%d", connectionID, remoteDir, len(localPaths))
	return a.connections.UploadFiles(a.callContext(), connectionID, remoteDir, localPaths)
}

func (a *App) PickDownloadDirectory() (string, error) {
	runtime.LogInfo(a.callContext(), "PickDownloadDirectory called")
	return runtime.OpenDirectoryDialog(a.callContext(), runtime.OpenDialogOptions{
		Title: "选择下载目录",
	})
}

func (a *App) PathExists(path string) (bool, error) {
	return appsvc.PathExists(path)
}

func (a *App) DownloadFile(connectionID string, remotePath string, localDir string, bytesTotal int64) (domain.TransferTask, error) {
	runtime.LogInfof(a.callContext(), "DownloadFile called: connection=%s remotePath=%s localDir=%s bytesTotal=%d", connectionID, remotePath, localDir, bytesTotal)
	return a.connections.DownloadFile(a.callContext(), connectionID, remotePath, localDir, bytesTotal)
}

func (a *App) ListTransferTasks() ([]domain.TransferTask, error) {
	return a.connections.ListTransferTasks(a.callContext())
}

func (a *App) CancelTransferTask(id string) error {
	runtime.LogInfof(a.callContext(), "CancelTransferTask called: id=%s", id)
	return a.connections.CancelTransferTask(a.callContext(), id)
}

func (a *App) callContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
