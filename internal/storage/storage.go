// Package storage defines the seams the API and worker depend on for
// metadata persistence and object storage, so implementations (SQLite, S3,
// or test fakes) can be swapped without touching business logic.
package storage

import (
	"context"
	"errors"

	"github.com/haegenq/securelink/internal/domain"
)

// ErrNotFound is returned by FileRepository.Get when no file matches the ID.
var ErrNotFound = errors.New("file not found")

// FileRepository persists and retrieves file metadata.
type FileRepository interface {
	Create(ctx context.Context, f *domain.File) error
	Get(ctx context.Context, id string) (*domain.File, error)
	UpdateStatus(ctx context.Context, id string, status domain.Status) error
}

// Uploader generates short-lived presigned URLs for direct client-to-S3
// transfer, and computes the object key for a file.
type Uploader interface {
	ObjectKey(fileID, filename string) string
	PresignPutURL(ctx context.Context, objectKey, contentType string) (url string, expiresInSeconds int, err error)
	PresignGetURL(ctx context.Context, objectKey string) (url string, expiresInSeconds int, err error)
}
