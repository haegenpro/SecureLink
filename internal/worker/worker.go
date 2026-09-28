// Package worker consumes S3-object-created events from SQS and drives file
// metadata through pending -> processing -> processed/failed, safely under
// duplicate delivery.
package worker

import (
	"context"
	"errors"
	"log"

	"github.com/haegenq/securelink/internal/domain"
	"github.com/haegenq/securelink/internal/queue"
	"github.com/haegenq/securelink/internal/storage"
)

// Worker ties a queue.Consumer, storage.FileRepository, and Processor
// together into the idempotent processing loop described in Spec.md
// section 5.
type Worker struct {
	consumer  queue.Consumer
	repo      storage.FileRepository
	processor Processor
	logger    *log.Logger
}

// New builds a Worker. logger may be nil, in which case log.Default() is used.
func New(consumer queue.Consumer, repo storage.FileRepository, processor Processor, logger *log.Logger) *Worker {
	if logger == nil {
		logger = log.Default()
	}
	if processor == nil {
		processor = NoopProcessor{}
	}
	return &Worker{consumer: consumer, repo: repo, processor: processor, logger: logger}
}

// Run polls the queue until ctx is canceled.
func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := w.ProcessOnce(ctx); err != nil {
			w.logger.Printf("receive messages: %v", err)
		}
	}
}

// ProcessOnce performs a single receive-and-process cycle. It is exported so
// tests can drive exactly one iteration deterministically.
func (w *Worker) ProcessOnce(ctx context.Context) error {
	msgs, err := w.consumer.ReceiveMessages(ctx)
	if err != nil {
		return err
	}
	for _, m := range msgs {
		w.handleMessage(ctx, m)
	}
	return nil
}

func (w *Worker) handleMessage(ctx context.Context, msg queue.Message) {
	events, err := queue.ParseS3Event(msg.Body)
	if err != nil {
		// A malformed body will never become valid on redelivery, so treat it
		// as a poison message: log it and acknowledge so it doesn't loop
		// forever without a dead-letter queue in front of it.
		w.logger.Printf("malformed event, discarding message: %v", err)
		w.deleteMessage(ctx, msg.ReceiptHandle)
		return
	}

	for _, ev := range events {
		if err := w.handleEvent(ctx, ev); err != nil {
			w.logger.Printf("transient failure processing file %s, leaving message for retry: %v", ev.FileID, err)
			return
		}
	}
	w.deleteMessage(ctx, msg.ReceiptHandle)
}

// handleEvent implements the idempotent state machine for a single S3
// object-created event. A non-nil return means "transient, leave the
// message for redelivery"; permanent failures are recorded on the file and
// resolved to a nil return so the caller acknowledges the message.
func (w *Worker) handleEvent(ctx context.Context, ev queue.ObjectEvent) error {
	f, err := w.repo.Get(ctx, ev.FileID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			w.logger.Printf("event references unknown file %s, treating as permanent failure", ev.FileID)
			return nil
		}
		return err
	}

	if f.Status == domain.StatusProcessed {
		w.logger.Printf("file %s already processed, skipping (idempotent)", ev.FileID)
		return nil
	}

	if err := w.repo.UpdateStatus(ctx, ev.FileID, domain.StatusProcessing); err != nil {
		return err
	}

	if err := w.processor.Process(ctx, f); err != nil {
		var perm *PermanentError
		if errors.As(err, &perm) {
			w.logger.Printf("file %s failed permanently: %v", ev.FileID, perm)
			return w.repo.UpdateStatus(ctx, ev.FileID, domain.StatusFailed)
		}
		return err
	}

	return w.repo.UpdateStatus(ctx, ev.FileID, domain.StatusProcessed)
}

func (w *Worker) deleteMessage(ctx context.Context, receiptHandle string) {
	if err := w.consumer.DeleteMessage(ctx, receiptHandle); err != nil {
		w.logger.Printf("delete message: %v", err)
	}
}
