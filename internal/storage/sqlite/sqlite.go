// Package sqlite implements storage.FileRepository backed by a SQLite file.
// SQLite (via a pure-Go driver) is chosen over an in-memory map so metadata
// survives process restarts, without requiring a separate database server.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/haegenq/securelink/internal/domain"
	"github.com/haegenq/securelink/internal/storage"
)

const schema = `
CREATE TABLE IF NOT EXISTS files (
	id           TEXT PRIMARY KEY,
	filename     TEXT NOT NULL,
	content_type TEXT NOT NULL,
	object_key   TEXT NOT NULL,
	size         INTEGER NOT NULL DEFAULT 0,
	status       TEXT NOT NULL,
	created_at   TEXT NOT NULL,
	updated_at   TEXT NOT NULL
);
`

// Store is a SQLite-backed storage.FileRepository.
type Store struct {
	db *sql.DB
}

// Open creates (if needed) the parent directory and the SQLite database at
// path, applies the schema, and returns a ready-to-use Store.
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite handles a single writer at a time; cap the pool so callers get
	// serialized, predictable access instead of "database is locked" errors.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	return &Store{db: db}, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

var _ storage.FileRepository = (*Store)(nil)

func (s *Store) Create(ctx context.Context, f *domain.File) error {
	now := time.Now().UTC()
	f.CreatedAt = now
	f.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO files (id, filename, content_type, object_key, size, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.Filename, f.ContentType, f.ObjectKey, f.Size, string(f.Status),
		f.CreatedAt.Format(time.RFC3339Nano), f.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert file: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id string) (*domain.File, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, filename, content_type, object_key, size, status, created_at, updated_at
		FROM files WHERE id = ?`, id)

	var f domain.File
	var status, createdAt, updatedAt string
	if err := row.Scan(&f.ID, &f.Filename, &f.ContentType, &f.ObjectKey, &f.Size, &status, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("query file: %w", err)
	}
	f.Status = domain.Status(status)
	f.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	f.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return &f, nil
}

func (s *Store) UpdateStatus(ctx context.Context, id string, status domain.Status) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE files SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}
