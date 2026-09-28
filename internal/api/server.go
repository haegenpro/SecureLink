// Package api implements the HTTP layer: routing, request/response DTOs,
// and translation between domain errors and HTTP status codes. Business
// logic (validation, ID/key generation) is delegated to internal/domain and
// storage implementations so handlers stay thin.
package api

import (
	"log"
	"net/http"

	"github.com/haegenq/securelink/internal/storage"
)

// Server wires the HTTP handlers to their storage dependencies.
type Server struct {
	repo     storage.FileRepository
	uploader storage.Uploader
	logger   *log.Logger
}

// NewServer builds a Server. logger may be nil, in which case log.Default() is used.
func NewServer(repo storage.FileRepository, uploader storage.Uploader, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.Default()
	}
	return &Server{repo: repo, uploader: uploader, logger: logger}
}

// Routes returns the configured HTTP handler for the API.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /files", s.handleCreateFile)
	mux.HandleFunc("GET /files/{id}", s.handleGetFile)
	mux.HandleFunc("GET /files/{id}/download", s.handleDownloadFile)
	return mux
}
