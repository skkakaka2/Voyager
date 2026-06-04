package credential

import (
	"context"

	keyring "github.com/zalando/go-keyring"
)

const ServiceName = "Voyager"

type KeyringStore struct {
	service string
}

func NewKeyringStore(service string) *KeyringStore {
	return &KeyringStore{service: service}
}

func (s *KeyringStore) Get(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return keyring.Get(s.service, key)
}

func (s *KeyringStore) Save(ctx context.Context, key string, password string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return keyring.Set(s.service, key, password)
}

func (s *KeyringStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return keyring.Delete(s.service, key)
}
