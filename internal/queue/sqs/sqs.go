// Package sqs implements queue.Consumer using the AWS SDK v2 SQS client.
package sqs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/haegenq/securelink/internal/queue"
)

// Config controls how the SQS client connects and polls.
type Config struct {
	Region      string
	QueueURL    string
	EndpointURL string // non-empty overrides the endpoint, e.g. LocalStack
	WaitTime    time.Duration
	MaxMessages int32
}

// Client is the real, AWS-SDK-backed queue.Consumer.
type Client struct {
	cfg Config
	sqs *sqs.Client
}

var _ queue.Consumer = (*Client)(nil)

// New builds a Client, loading AWS credentials via the standard SDK provider
// chain (env vars, shared config, IAM role).
func New(ctx context.Context, cfg Config) (*Client, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	return newClient(cfg, awsCfg), nil
}

// NewWithStaticCredentials is a convenience constructor for local dev against
// LocalStack, which accepts any non-empty access key/secret.
func NewWithStaticCredentials(ctx context.Context, cfg Config, accessKeyID, secretAccessKey string) (*Client, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	return newClient(cfg, awsCfg), nil
}

func newClient(cfg Config, awsCfg aws.Config) *Client {
	client := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		if cfg.EndpointURL != "" {
			o.BaseEndpoint = aws.String(cfg.EndpointURL)
		}
	})
	return &Client{cfg: cfg, sqs: client}
}

func (c *Client) ReceiveMessages(ctx context.Context) ([]queue.Message, error) {
	out, err := c.sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.cfg.QueueURL),
		MaxNumberOfMessages: c.cfg.MaxMessages,
		WaitTimeSeconds:     int32(c.cfg.WaitTime.Seconds()),
	})
	if err != nil {
		return nil, fmt.Errorf("receive message: %w", err)
	}

	msgs := make([]queue.Message, 0, len(out.Messages))
	for _, m := range out.Messages {
		msgs = append(msgs, queue.Message{
			ReceiptHandle: aws.ToString(m.ReceiptHandle),
			Body:          aws.ToString(m.Body),
		})
	}
	return msgs, nil
}

func (c *Client) DeleteMessage(ctx context.Context, receiptHandle string) error {
	_, err := c.sqs.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.cfg.QueueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	return nil
}
