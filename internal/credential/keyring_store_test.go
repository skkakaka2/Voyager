package credential

import (
	"context"
	"testing"

	keyring "github.com/zalando/go-keyring"
)

func TestKeyringStoreSavesGetsAndDeletesPassword(t *testing.T) {
	keyring.MockInit()
	store := NewKeyringStore("VoyagerTest")
	ctx := context.Background()

	if err := store.Save(ctx, "connection-id", "super-secret"); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	password, err := store.Get(ctx, "connection-id")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if password != "super-secret" {
		t.Fatalf("expected stored password, got %q", password)
	}

	if err := store.Delete(ctx, "connection-id"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := store.Get(ctx, "connection-id"); err == nil {
		t.Fatal("expected deleted password to be missing")
	}
}
