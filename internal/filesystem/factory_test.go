package filesystem

import (
	"testing"

	"voyager/internal/domain"
)

func TestFactoryCreatesFTPAdapter(t *testing.T) {
	adapter, err := NewFactory().New(domain.ConnectionInput{
		Protocol: domain.ProtocolFTP,
		Host:     "ftp.example.com",
		Username: "ada",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if adapter == nil {
		t.Fatal("expected FTP adapter")
	}
}

func TestFactoryCreatesSMBAdapter(t *testing.T) {
	adapter, err := NewFactory().New(domain.ConnectionInput{
		Protocol: domain.ProtocolSMB,
		Host:     "nas.local",
		Share:    "public",
		Username: "ada",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if adapter == nil {
		t.Fatal("expected SMB adapter")
	}
}
