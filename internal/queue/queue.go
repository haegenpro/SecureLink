// Package queue wraps SQS message receive/delete behind an interface the
// worker depends on, and parses the S3 "ObjectCreated" event notifications
// that S3 delivers to the queue.
package queue

import "context"

// Message is a single SQS message: enough for the worker to process the
// body and later acknowledge (delete) it.
type Message struct {
	ReceiptHandle string
	Body          string
}

// Consumer receives and acknowledges SQS messages. Deleting a message is the
// SQS acknowledgment; a message that is never deleted becomes visible again
// after the queue's visibility timeout, which is what gives the worker
// automatic retries for free.
type Consumer interface {
	ReceiveMessages(ctx context.Context) ([]Message, error)
	DeleteMessage(ctx context.Context, receiptHandle string) error
}
