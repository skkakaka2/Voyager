package ftp

import (
	"testing"

	"voyager/internal/domain"
)

func TestNewAdapterBuildsDefaultAddressAndRootPath(t *testing.T) {
	adapter := NewAdapter(domain.ConnectionInput{
		Host:     "ftp.example.com",
		RootPath: "/public/docs/",
		Username: "ada",
		Password: "secret",
	})

	if adapter.address != "ftp.example.com:21" {
		t.Fatalf("expected default FTP address, got %q", adapter.address)
	}
	if adapter.rootPath != "/public/docs" {
		t.Fatalf("expected normalized root path, got %q", adapter.rootPath)
	}
	if adapter.remotePath("/reports/weekly.txt") != "/public/docs/reports/weekly.txt" {
		t.Fatalf("expected path under root, got %q", adapter.remotePath("/reports/weekly.txt"))
	}
}

func TestNewAdapterPreservesConfiguredPort(t *testing.T) {
	adapter := NewAdapter(domain.ConnectionInput{
		Host: "ftp.example.com",
		Port: 2121,
	})

	if adapter.address != "ftp.example.com:2121" {
		t.Fatalf("expected configured FTP address, got %q", adapter.address)
	}
}
