package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/haegenq/securelink/internal/domain"
	"github.com/haegenq/securelink/internal/storage"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func (s *Server) handleCreateFile(w http.ResponseWriter, r *http.Request) {
	var req createFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := domain.ValidateFilename(req.Filename); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := domain.ValidateContentType(req.ContentType); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Size < 0 {
		writeError(w, http.StatusBadRequest, "size must not be negative")
		return
	}

	id, err := newFileID()
	if err != nil {
		s.logger.Printf("generate file id: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	objectKey := s.uploader.ObjectKey(id, req.Filename)

	f := &domain.File{
		ID:          id,
		Filename:    req.Filename,
		ContentType: req.ContentType,
		ObjectKey:   objectKey,
		Size:        req.Size,
		Status:      domain.StatusPending,
	}

	ctx := r.Context()
	if err := s.repo.Create(ctx, f); err != nil {
		s.logger.Printf("create file %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	uploadURL, expiresIn, err := s.uploader.PresignPutURL(ctx, objectKey, req.ContentType)
	if err != nil {
		s.logger.Printf("presign put for file %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Never log the URL itself: it embeds a valid, time-limited credential.
	s.logger.Printf("created file %s (upload URL issued, expires_in=%ds)", id, expiresIn)

	writeJSON(w, http.StatusCreated, createFileResponse{
		FileID:    id,
		UploadURL: uploadURL,
		ExpiresIn: expiresIn,
	})
}

func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	f, err := s.repo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		s.logger.Printf("get file %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, fileStatusResponse{
		ID:          f.ID,
		Status:      string(f.Status),
		Filename:    f.Filename,
		ContentType: f.ContentType,
		Size:        f.Size,
		CreatedAt:   f.CreatedAt,
	})
}

func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	f, err := s.repo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		s.logger.Printf("get file %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	url, expiresIn, err := s.uploader.PresignGetURL(r.Context(), f.ObjectKey)
	if err != nil {
		s.logger.Printf("presign get for file %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	s.logger.Printf("issued download URL for file %s (expires_in=%ds)", id, expiresIn)

	writeJSON(w, http.StatusOK, downloadResponse{DownloadURL: url, ExpiresIn: expiresIn})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
