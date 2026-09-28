// Package s3 implements storage.Uploader using presigned S3 PUT/GET URLs, so
// clients transfer file bytes directly to/from S3 and never through the API
// process.
package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/haegenq/securelink/internal/domain"
	"github.com/haegenq/securelink/internal/storage"
)

// Config controls how the presign client connects to S3 or an S3-compatible
// endpoint such as LocalStack.
type Config struct {
	Region         string
	Bucket         string
	EndpointURL    string // non-empty overrides the endpoint, e.g. LocalStack
	UploadURLTTL   time.Duration
	DownloadURLTTL time.Duration
}

// PresignUploader is the real, AWS-SDK-backed storage.Uploader.
type PresignUploader struct {
	cfg     Config
	presign *s3.PresignClient
}

var _ storage.Uploader = (*PresignUploader)(nil)

// New builds a PresignUploader from the given Config, loading AWS credentials
// via the standard SDK provider chain (env vars, shared config, IAM role).
func New(ctx context.Context, cfg Config) (*PresignUploader, error) {
	loadOpts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.EndpointURL != "" {
			o.BaseEndpoint = aws.String(cfg.EndpointURL)
			// LocalStack and most S3-compatible endpoints need path-style
			// addressing rather than virtual-hosted buckets.
			o.UsePathStyle = true
		}
	})

	return &PresignUploader{
		cfg:     cfg,
		presign: s3.NewPresignClient(client),
	}, nil
}

// NewWithStaticCredentials is a convenience constructor for local dev against
// LocalStack, which accepts any non-empty access key/secret.
func NewWithStaticCredentials(ctx context.Context, cfg Config, accessKeyID, secretAccessKey string) (*PresignUploader, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.EndpointURL != "" {
			o.BaseEndpoint = aws.String(cfg.EndpointURL)
			o.UsePathStyle = true
		}
	})

	return &PresignUploader{
		cfg:     cfg,
		presign: s3.NewPresignClient(client),
	}, nil
}

func (u *PresignUploader) ObjectKey(fileID, filename string) string {
	return domain.SanitizedObjectKey(fileID, filename)
}

func (u *PresignUploader) PresignPutURL(ctx context.Context, objectKey, contentType string) (string, int, error) {
	ttl := u.cfg.UploadURLTTL
	req, err := u.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(u.cfg.Bucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", 0, fmt.Errorf("presign put: %w", err)
	}
	return req.URL, int(ttl.Seconds()), nil
}

func (u *PresignUploader) PresignGetURL(ctx context.Context, objectKey string) (string, int, error) {
	ttl := u.cfg.DownloadURLTTL
	req, err := u.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(u.cfg.Bucket),
		Key:    aws.String(objectKey),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", 0, fmt.Errorf("presign get: %w", err)
	}
	return req.URL, int(ttl.Seconds()), nil
}
