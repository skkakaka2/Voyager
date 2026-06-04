package filesystem

import (
	"context"

	"voyager/internal/domain"
)

type ProgressFunc = func(bytesDone int64)

type Adapter interface {
	Test(ctx context.Context) error
	List(ctx context.Context, path string) ([]domain.RemoteEntry, error)
	Mkdir(ctx context.Context, path string) error
	Rename(ctx context.Context, oldPath string, newPath string) error
	Delete(ctx context.Context, path string) error
	Upload(ctx context.Context, localPath string, remotePath string, progress ProgressFunc) error
	Download(ctx context.Context, remotePath string, localPath string, progress ProgressFunc) error
}
