// Package config loads runtime configuration from environment variables,
// with sane defaults for local development.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds every environment-derived setting used by the API and worker.
type Config struct {
	APIAddr string

	DBPath string

	AWSRegion      string
	AWSEndpointURL string // non-empty targets LocalStack or another S3/SQS-compatible endpoint
	S3Bucket       string
	SQSQueueURL    string

	UploadURLTTL   time.Duration
	DownloadURLTTL time.Duration

	SQSWaitTime    time.Duration
	SQSMaxMessages int32
}

// Load reads Config from the environment, falling back to local-dev defaults
// for anything unset.
func Load() Config {
	return Config{
		APIAddr: getEnv("API_ADDR", ":8080"),

		DBPath: getEnv("DB_PATH", "./data/securelink.db"),

		AWSRegion:      getEnv("AWS_REGION", "us-east-1"),
		AWSEndpointURL: getEnv("AWS_ENDPOINT_URL", ""),
		S3Bucket:       getEnv("S3_BUCKET", "securelink-files"),
		SQSQueueURL:    getEnv("SQS_QUEUE_URL", ""),

		UploadURLTTL:   getEnvSeconds("UPLOAD_URL_TTL_SECONDS", 300),
		DownloadURLTTL: getEnvSeconds("DOWNLOAD_URL_TTL_SECONDS", 300),

		SQSWaitTime:    getEnvSeconds("SQS_WAIT_TIME_SECONDS", 10),
		SQSMaxMessages: int32(getEnvInt("SQS_MAX_MESSAGES", 5)),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvSeconds(key string, fallbackSeconds int) time.Duration {
	return time.Duration(getEnvInt(key, fallbackSeconds)) * time.Second
}
