package smb

import (
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
