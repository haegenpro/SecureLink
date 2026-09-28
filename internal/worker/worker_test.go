package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"testing"

	"github.com/haegenq/securelink/internal/domain"
	"github.com/haegenq/securelink/internal/queue"
	"github.com/haegenq/securelink/internal/storage"
)

// fakeConsumer serves a fixed batch of messages once, then empty batches,
// and records which receipt handles were deleted.
type fakeConsumer struct {
	mu      sync.Mutex
	pending []queue.Message
	deleted []string
}

func (c *fakeConsumer) ReceiveMessages(ctx context.Context) ([]queue.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	msgs := c.pending
	c.pending = nil
	return msgs, nil
}

func (c *fakeConsumer) DeleteMessage(ctx context.Context, receiptHandle string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleted = append(c.deleted, receiptHandle)
	return nil
}

func (c *fakeConsumer) wasDeleted(handle string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, h := range c.deleted {
		if h == handle {
			return true
		}
	}
	return false
}

// fakeRepo is a minimal in-memory storage.FileRepository.
type fakeRepo struct {
	mu    sync.Mutex
	files map[string]*domain.File
}

func newFakeRepo(files ...*domain.File) *fakeRepo {
	r := &fakeRepo{files: make(map[string]*domain.File)}
	for _, f := range files {
		cp := *f
		r.files[f.ID] = &cp
	}
	return r
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

func (r *fakeRepo) statusOf(t *testing.T, id string) domain.Status {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.files[id]
	if !ok {
		t.Fatalf("file %s not found", id)
	}
	return f.Status
}

const eventBodyFor = `{"Records":[{"eventName":"ObjectCreated:Put","s3":{"bucket":{"name":"b"},"object":{"key":"%s/report.pdf","size":1024}}}]}`

func testLogger() *log.Logger {
	return log.New(nopWriter{}, "", 0)
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestProcessOnce_ValidEvent(t *testing.T) {
	repo := newFakeRepo(&domain.File{ID: "abc123", Status: domain.StatusPending})
	consumer := &fakeConsumer{pending: []queue.Message{{ReceiptHandle: "h1", Body: fmt.Sprintf(eventBodyFor, "abc123")}}}
	w := New(consumer, repo, NoopProcessor{}, testLogger())

	if err := w.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("ProcessOnce: %v", err)
	}

	if got := repo.statusOf(t, "abc123"); got != domain.StatusProcessed {
		t.Fatalf("status = %q, want processed", got)
	}
	if !consumer.wasDeleted("h1") {
		t.Fatal("expected message to be acknowledged (deleted)")
	}
}

func TestProcessOnce_DuplicateEventIsIdempotent(t *testing.T) {
	repo := newFakeRepo(&domain.File{ID: "abc123", Status: domain.StatusProcessed})
	consumer := &fakeConsumer{pending: []queue.Message{{ReceiptHandle: "h1", Body: fmt.Sprintf(eventBodyFor, "abc123")}}}
	w := New(consumer, repo, alwaysFailProcessor{}, testLogger())

	if err := w.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("ProcessOnce: %v", err)
	}

	// Already-processed files must not be reprocessed (the failing
	// processor would otherwise flip status to failed), and the duplicate
	// delivery must still be acknowledged.
	if got := repo.statusOf(t, "abc123"); got != domain.StatusProcessed {
		t.Fatalf("status = %q, want processed (unchanged)", got)
	}
	if !consumer.wasDeleted("h1") {
		t.Fatal("expected duplicate message to be acknowledged")
	}
}

func TestProcessOnce_MalformedEventIsSafe(t *testing.T) {
	repo := newFakeRepo()
	consumer := &fakeConsumer{pending: []queue.Message{{ReceiptHandle: "h1", Body: "not json"}}}
	w := New(consumer, repo, NoopProcessor{}, testLogger())

	if err := w.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("ProcessOnce: %v", err)
	}
	if !consumer.wasDeleted("h1") {
		t.Fatal("expected malformed message to be acknowledged (poison message)")
	}
}

type alwaysFailProcessor struct{}

func (alwaysFailProcessor) Process(ctx context.Context, f *domain.File) error {
	return errors.New("boom")
}

type permanentFailProcessor struct{}

func (permanentFailProcessor) Process(ctx context.Context, f *domain.File) error {
	return Permanent(errors.New("unsupported content"))
}

func TestProcessOnce_TransientFailureDoesNotAck(t *testing.T) {
	repo := newFakeRepo(&domain.File{ID: "abc123", Status: domain.StatusPending})
	consumer := &fakeConsumer{pending: []queue.Message{{ReceiptHandle: "h1", Body: fmt.Sprintf(eventBodyFor, "abc123")}}}
	w := New(consumer, repo, alwaysFailProcessor{}, testLogger())

	if err := w.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("ProcessOnce: %v", err)
	}

	if consumer.wasDeleted("h1") {
		t.Fatal("transient failure must not acknowledge the message")
	}
	if got := repo.statusOf(t, "abc123"); got != domain.StatusProcessing {
		t.Fatalf("status = %q, want processing (left for retry)", got)
	}
}

func TestProcessOnce_PermanentFailureMarksFailedAndAcks(t *testing.T) {
	repo := newFakeRepo(&domain.File{ID: "abc123", Status: domain.StatusPending})
	consumer := &fakeConsumer{pending: []queue.Message{{ReceiptHandle: "h1", Body: fmt.Sprintf(eventBodyFor, "abc123")}}}
	w := New(consumer, repo, permanentFailProcessor{}, testLogger())

	if err := w.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("ProcessOnce: %v", err)
	}

	if got := repo.statusOf(t, "abc123"); got != domain.StatusFailed {
		t.Fatalf("status = %q, want failed", got)
	}
	if !consumer.wasDeleted("h1") {
		t.Fatal("expected permanently-failed message to be acknowledged")
	}
}
