// Command api runs the SecureLink HTTP API server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/haegenq/securelink/internal/api"
	"github.com/haegenq/securelink/internal/config"
	"github.com/haegenq/securelink/internal/storage/s3"
	"github.com/haegenq/securelink/internal/storage/sqlite"
)

func main() {
	logger := log.New(os.Stdout, "[api] ", log.LstdFlags)

	cfg := config.Load()

	repo, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		logger.Fatalf("open database: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	uploader, err := newUploader(ctx, cfg)
	if err != nil {
		logger.Fatalf("configure S3 uploader: %v", err)
	}

	server := api.NewServer(repo, uploader, logger)

	httpServer := &http.Server{
		Addr:    cfg.APIAddr,
		Handler: server.Routes(),
	}

	go func() {
		logger.Printf("listening on %s", cfg.APIAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Print("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Printf("shutdown: %v", err)
	}
}

func newUploader(ctx context.Context, cfg config.Config) (*s3.PresignUploader, error) {
	s3Cfg := s3.Config{
		Region:         cfg.AWSRegion,
		Bucket:         cfg.S3Bucket,
		EndpointURL:    cfg.AWSEndpointURL,
		UploadURLTTL:   cfg.UploadURLTTL,
		DownloadURLTTL: cfg.DownloadURLTTL,
	}

	// LocalStack accepts any non-empty static credentials; using them when an
	// endpoint override is set avoids requiring real AWS credentials locally.
	if cfg.AWSEndpointURL != "" {
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		if accessKey != "" && secretKey != "" {
			return s3.NewWithStaticCredentials(ctx, s3Cfg, accessKey, secretKey)
		}
	}
	return s3.New(ctx, s3Cfg)
}
