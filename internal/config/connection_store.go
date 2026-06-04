package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"voyager/internal/domain"
)

type ConnectionStore struct {
	path string
}

type connectionFile struct {
	Connections []domain.ConnectionProfile `json:"connections"`
}

func NewConnectionStore(path string) *ConnectionStore {
	return &ConnectionStore{path: path}
}

func NewDefaultConnectionStore() (*ConnectionStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return NewConnectionStore(filepath.Join(dir, "Voyager", "connections.json")), nil
}

func (s *ConnectionStore) Load() ([]domain.ConnectionProfile, error) {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []domain.ConnectionProfile{}, nil
	}
	if err != nil {
		return nil, err
	}

	var file connectionFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, err
	}
	if file.Connections == nil {
		return []domain.ConnectionProfile{}, nil
	}
	return file.Connections, nil
}

func (s *ConnectionStore) Save(connections []domain.ConnectionProfile) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(connectionFile{Connections: connections}, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
