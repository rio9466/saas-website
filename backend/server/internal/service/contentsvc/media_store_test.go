package contentsvc

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestLocalDiskMediaStoreRoundTrip(t *testing.T) {
	t.Parallel()
	store, err := NewLocalDiskMediaStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalDiskMediaStore: %v", err)
	}
	ctx := context.Background()
	payload := []byte("media-bytes")
	if err := store.Put(ctx, "abc.png", bytes.NewReader(payload)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	file, err := store.Open("abc.png")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer file.Close()
	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("Open content = %q, want %q", got, payload)
	}
	if file.Size() != int64(len(payload)) {
		t.Fatalf("Size = %d, want %d", file.Size(), len(payload))
	}
	if err := store.Delete("abc.png"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Open("abc.png"); err == nil {
		t.Fatal("Open after Delete must fail")
	}
	if err := store.Delete("abc.png"); err != nil {
		t.Fatalf("Delete missing object = %v, want nil", err)
	}
}

func TestLocalDiskMediaStoreRejectsTraversal(t *testing.T) {
	t.Parallel()
	store, err := NewLocalDiskMediaStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalDiskMediaStore: %v", err)
	}
	ctx := context.Background()
	for _, name := range []string{"", ".", "..", "a/b", "../escape", `a\b`} {
		if err := store.Put(ctx, name, bytes.NewReader([]byte("x"))); err == nil {
			t.Fatalf("Put(%q) must be rejected", name)
		}
		if _, err := store.Open(name); err == nil {
			t.Fatalf("Open(%q) must be rejected", name)
		}
		if err := store.Delete(name); err == nil {
			t.Fatalf("Delete(%q) must be rejected", name)
		}
	}
}
