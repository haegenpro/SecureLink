// Package domain holds the core File entity and validation rules.
// It has no dependency on AWS, SQLite, or HTTP so it stays trivially unit-testable.
package domain

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
)

// Status represents the lifecycle of a file's processing.
type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusProcessed  Status = "processed"
	StatusFailed     Status = "failed"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusProcessing, StatusProcessed, StatusFailed:
		return true
	default:
		return false
	}
}

// File is the metadata record for a single file transfer.
type File struct {
	ID          string
	Filename    string
	ContentType string
	ObjectKey   string
	Size        int64
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var (
	ErrInvalidFilename    = errors.New("filename is invalid")
	ErrInvalidContentType = errors.New("content_type is invalid")
	ErrFilenameTooLong    = errors.New("filename is too long")
)

const maxFilenameLength = 255

// ValidateFilename rejects empty names, path separators, traversal sequences,
// and anything that isn't a plain base filename. This is what keeps a
// user-controlled filename from ever being used to construct an unsafe path.
func ValidateFilename(name string) error {
	if name == "" {
		return ErrInvalidFilename
	}
	if len(name) > maxFilenameLength {
		return ErrFilenameTooLong
	}
	if strings.ContainsAny(name, "/\\") {
		return ErrInvalidFilename
	}
	if name == "." || name == ".." {
		return ErrInvalidFilename
	}
	if filepath.Base(name) != name {
		return ErrInvalidFilename
	}
	return nil
}

// ValidateContentType requires a simple "type/subtype" MIME-ish string and
// rejects anything containing control characters or separators that would
// look out of place in an HTTP header.
func ValidateContentType(ct string) error {
	if ct == "" {
		return ErrInvalidContentType
	}
	if len(ct) > 255 {
		return ErrInvalidContentType
	}
	parts := strings.SplitN(ct, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ErrInvalidContentType
	}
	for _, r := range ct {
		if r < 0x20 || r == 0x7f {
			return ErrInvalidContentType
		}
	}
	return nil
}

// SanitizedObjectKey builds the S3 object key for a file, namespaced by ID so
// two uploads can never collide even if filenames match.
func SanitizedObjectKey(id, filename string) string {
	return id + "/" + filename
}
