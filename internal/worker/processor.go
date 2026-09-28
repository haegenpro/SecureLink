package worker

import (
	"context"

	"github.com/haegenq/securelink/internal/domain"
)

// Processor performs the actual per-file work once a file's upload has been
// confirmed via its S3 event. Returning an error wrapped with Permanent
// marks the file as permanently failed; any other error is treated as
// transient and left for SQS to redeliver.
type Processor interface {
	Process(ctx context.Context, f *domain.File) error
}

// NoopProcessor is the project's intentionally lightweight "processing"
// step: it does no real transformation, virus scanning, or content
// inspection (explicitly out of scope per Spec.md), and always succeeds.
// It exists to demonstrate the pipeline's shape without overclaiming
// functionality that was never implemented.
type NoopProcessor struct{}

func (NoopProcessor) Process(ctx context.Context, f *domain.File) error {
	return nil
}
