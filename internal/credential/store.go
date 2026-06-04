package credential

import (
	"context"
	"errors"
)

var ErrUnavailable = errors.New("system keyring unavailable")

type Store interface {
	Get(ctx context.Context, key string) (string, error)
	Save(ctx context.Context, key string, password string) error
	Delete(ctx context.Context, key string) error
}

type UnavailableStore struct{}

func (UnavailableStore) Get(context.Context, string) (string, error) {
	return "", ErrUnavailable
}

func (UnavailableStore) Save(context.Context, string, string) error {
	return ErrUnavailable
}

func (UnavailableStore) Delete(context.Context, string) error {
	return nil
}
