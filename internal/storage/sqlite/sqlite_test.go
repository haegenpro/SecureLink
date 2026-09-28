package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/haegenq/securelink/internal/domain"
	"github.com/haegenq/securelink/internal/storage"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndGet(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	f := &domain.File{
		ID:          "abc123",
		Filename:    "report.pdf",
		ContentType: "application/pdf",
		ObjectKey:   "abc123/report.pdf",
		Status:      domain.StatusPending,
	}
	if err := s.Create(ctx, f); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.Get(ctx, "abc123")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Filename != f.Filename || got.Status != domain.StatusPending {
		t.Fatalf("Get returned %+v, want filename %q status %q", got, f.Filename, domain.StatusPending)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatal("expected timestamps to be set")
	}
}

func TestGetNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get(context.Background(), "does-not-exist")
	if err != storage.ErrNotFound {
		t.Fatalf("Get error = %v, want storage.ErrNotFound", err)
	}
}

func TestUpdateStatus(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	f := &domain.File{ID: "id1", Filename: "a.txt", ContentType: "text/plain", ObjectKey: "id1/a.txt", Status: domain.StatusPending}
	if err := s.Create(ctx, f); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.UpdateStatus(ctx, "id1", domain.StatusProcessed); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, err := s.Get(ctx, "id1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != domain.StatusProcessed {
		t.Fatalf("Status = %q, want %q", got.Status, domain.StatusProcessed)
	}
}

func TestUpdateStatusNotFound(t *testing.T) {
	s := newTestStore(t)
	err := s.UpdateStatus(context.Background(), "missing", domain.StatusProcessed)
	if err != storage.ErrNotFound {
		t.Fatalf("UpdateStatus error = %v, want storage.ErrNotFound", err)
	}
}
