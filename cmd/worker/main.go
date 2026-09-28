// Command worker consumes S3 object-created events from SQS and processes
// them into file status updates.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/haegenq/securelink/internal/config"
	"github.com/haegenq/securelink/internal/queue/sqs"
	"github.com/haegenq/securelink/internal/storage/sqlite"
	"github.com/haegenq/securelink/internal/worker"
)

func main() {
	logger := log.New(os.Stdout, "[worker] ", log.LstdFlags)

	cfg := config.Load()

	repo, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		logger.Fatalf("open database: %v", err)
	}
	defer repo.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	consumer, err := newConsumer(ctx, cfg)
	if err != nil {
		logger.Fatalf("configure SQS consumer: %v", err)
	}

	w := worker.New(consumer, repo, worker.NoopProcessor{}, logger)

	logger.Printf("polling queue %s", cfg.SQSQueueURL)
	w.Run(ctx)
	logger.Print("shutting down")
}

func newConsumer(ctx context.Context, cfg config.Config) (*sqs.Client, error) {
	sqsCfg := sqs.Config{
		Region:      cfg.AWSRegion,
		QueueURL:    cfg.SQSQueueURL,
		EndpointURL: cfg.AWSEndpointURL,
		WaitTime:    cfg.SQSWaitTime,
		MaxMessages: cfg.SQSMaxMessages,
	}

	if cfg.AWSEndpointURL != "" {
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		if accessKey != "" && secretKey != "" {
			return sqs.NewWithStaticCredentials(ctx, sqsCfg, accessKey, secretKey)
		}
	}
	return sqs.New(ctx, sqsCfg)
}
