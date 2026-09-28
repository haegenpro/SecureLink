package api

import (
	"context"
	"sync"

	"github.com/haegenq/securelink/internal/domain"
	"github.com/haegenq/securelink/internal/storage"
)

// fakeRepo is an in-memory storage.FileRepository for handler tests.
type fakeRepo struct {
	mu    sync.Mutex
	files map[string]*domain.File
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{files: make(map[string]*domain.File)}
}

func (r *fakeRepo) Create(ctx context.Context, f *domain.File) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *f
	r.files[f.ID] = &cp
	return nil
}

func (r *fakeRepo) Get(ctx context.Context, id string) (*domain.File, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.files[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *f
	return &cp, nil
}

func (r *fakeRepo) UpdateStatus(ctx context.Context, id string, status domain.Status) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.files[id]
	if !ok {
		return storage.ErrNotFound
	}
	f.Status = status
	return nil
}

// fakeUploader is a storage.Uploader that never touches AWS.
type fakeUploader struct{}

func (fakeUploader) ObjectKey(fileID, filename string) string {
	return domain.SanitizedObjectKey(fileID, filename)
}

func (fakeUploader) PresignPutURL(ctx context.Context, objectKey, contentType string) (string, int, error) {
	return "https://example-bucket.s3.amazonaws.com/" + objectKey + "?put-signature=fake", 300, nil
}

func (fakeUploader) PresignGetURL(ctx context.Context, objectKey string) (string, int, error) {
	return "https://example-bucket.s3.amazonaws.com/" + objectKey + "?get-signature=fake", 300, nil
}
