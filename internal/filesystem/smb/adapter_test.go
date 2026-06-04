package smb

import (
	"context"
	"testing"

	"voyager/internal/domain"
)

func TestNewAdapterBuildsDefaultAddressShareAndRootPath(t *testing.T) {
	adapter := NewAdapter(domain.ConnectionInput{
		Host:     "nas.local",
		Share:    "/public/",
		RootPath: "/team/docs/",
		Username: "ada",
		Password: "secret",
	})

	if adapter.address != "nas.local:445" {
		t.Fatalf("expected default SMB address, got %q", adapter.address)
	}
	if adapter.shareName != "public" {
		t.Fatalf("expected normalized share name, got %q", adapter.shareName)
	}
	if adapter.rootPath != "team/docs" {
		t.Fatalf("expected normalized root path, got %q", adapter.rootPath)
	}
	if adapter.sharePath("/reports/weekly.txt") != "team/docs/reports/weekly.txt" {
		t.Fatalf("expected path under root, got %q", adapter.sharePath("/reports/weekly.txt"))
	}
}

func TestNewAdapterPreservesConfiguredPort(t *testing.T) {
	adapter := NewAdapter(domain.ConnectionInput{
		Host: "nas.local",
		Port: 1445,
	})

	if adapter.address != "nas.local:1445" {
		t.Fatalf("expected configured SMB address, got %q", adapter.address)
	}
}

func TestListRootReturnsServerSharesWhenShareIsEmpty(t *testing.T) {
	adapter := NewAdapter(domain.ConnectionInput{Host: "nas.local"})
	adapter.listShareNames = func(context.Context) ([]string, error) {
		return []string{"IPC$", "Photos", "file", "ADMIN$", "iso"}, nil
	}

	entries, err := adapter.List(context.Background(), "/")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	want := []domain.RemoteEntry{
		{Name: "Photos", Path: "/Photos", Type: domain.EntryDirectory},
		{Name: "file", Path: "/file", Type: domain.EntryDirectory},
		{Name: "iso", Path: "/iso", Type: domain.EntryDirectory},
	}
	if len(entries) != len(want) {
		t.Fatalf("expected %d share entries, got %#v", len(want), entries)
	}
	for i := range want {
		if entries[i].Name != want[i].Name || entries[i].Path != want[i].Path || entries[i].Type != want[i].Type {
			t.Fatalf("entry %d mismatch: want %#v, got %#v", i, want[i], entries[i])
		}
	}
}

func TestResolveSharePathUsesFirstSegmentWhenShareIsEmpty(t *testing.T) {
	adapter := NewAdapter(domain.ConnectionInput{Host: "nas.local"})

	shareName, sharePath, err := adapter.resolveSharePath("/Photos/trips/2026")
	if err != nil {
		t.Fatalf("resolveSharePath returned error: %v", err)
	}
	if shareName != "Photos" || sharePath != "trips/2026" {
		t.Fatalf("expected Photos + trips/2026, got %q + %q", shareName, sharePath)
	}

	shareName, sharePath, err = adapter.resolveSharePath("/Photos")
	if err != nil {
		t.Fatalf("resolveSharePath returned error: %v", err)
	}
	if shareName != "Photos" || sharePath != "" {
		t.Fatalf("expected Photos + empty path, got %q + %q", shareName, sharePath)
	}
}

func TestMutatingOperationsAtServerRootRequireShare(t *testing.T) {
	adapter := NewAdapter(domain.ConnectionInput{Host: "nas.local"})
	localFile := t.TempDir() + "/report.txt"

	tests := []struct {
		name string
		run  func() error
	}{
		{name: "mkdir", run: func() error { return adapter.Mkdir(context.Background(), "/") }},
		{name: "rename", run: func() error { return adapter.Rename(context.Background(), "/", "/renamed") }},
		{name: "delete", run: func() error { return adapter.Delete(context.Background(), "/") }},
		{name: "upload", run: func() error { return adapter.Upload(context.Background(), localFile, "/", nil) }},
		{name: "download", run: func() error { return adapter.Download(context.Background(), "/", localFile, nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || err.Error() != "请先进入一个 SMB 共享" {
				t.Fatalf("expected share-required error, got %v", err)
			}
		})
	}

	adapterWithRoot := NewAdapter(domain.ConnectionInput{Host: "nas.local", RootPath: "/team"})
	err := adapterWithRoot.Delete(context.Background(), "/Photos")
	if err == nil || err.Error() != "请先进入一个 SMB 共享" {
		t.Fatalf("expected share-required error for share entry with root path, got %v", err)
	}
}
