package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/haegenq/securelink/internal/domain"
)

func newTestServer() (*Server, *fakeRepo) {
	repo := newFakeRepo()
	s := NewServer(repo, fakeUploader{}, nil)
	return s, repo
}

func TestHealth(t *testing.T) {
	s, _ := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	s.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body healthResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status = %q, want ok", body.Status)
	}
}

func TestCreateFileInvalidJSON(t *testing.T) {
	s, _ := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/files", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()

	s.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateFileInvalidFilename(t *testing.T) {
	s, _ := newTestServer()
	body, _ := json.Marshal(createFileRequest{Filename: "../etc/passwd", ContentType: "text/plain"})
	req := httptest.NewRequest(http.MethodPost, "/files", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateFileSuccess(t *testing.T) {
	s, repo := newTestServer()
	body, _ := json.Marshal(createFileRequest{Filename: "report.pdf", ContentType: "application/pdf", Size: 1024})
	req := httptest.NewRequest(http.MethodPost, "/files", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", w.Code, w.Body.String())
	}
	var resp createFileResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.FileID == "" {
		t.Fatal("expected non-empty file_id")
	}
	if resp.UploadURL == "" {
		t.Fatal("expected non-empty upload_url")
	}
	if resp.ExpiresIn != 300 {
		t.Fatalf("expires_in = %d, want 300", resp.ExpiresIn)
	}

	stored, err := repo.Get(context.Background(), resp.FileID)
	if err != nil {
		t.Fatalf("expected file to be persisted: %v", err)
	}
	if stored.Status != domain.StatusPending {
		t.Fatalf("status = %q, want pending", stored.Status)
	}
}

func TestGetFileNotFound(t *testing.T) {
	s, _ := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/files/does-not-exist", nil)
	w := httptest.NewRecorder()

	s.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestGetFileSuccess(t *testing.T) {
	s, repo := newTestServer()
	ctx := context.Background()
	_ = repo.Create(ctx, &domain.File{
		ID: "abc123", Filename: "report.pdf", ContentType: "application/pdf",
		ObjectKey: "abc123/report.pdf", Size: 183920, Status: domain.StatusProcessed,
	})

	req := httptest.NewRequest(http.MethodGet, "/files/abc123", nil)
	w := httptest.NewRecorder()
	s.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp fileStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "processed" || resp.Filename != "report.pdf" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestDownloadFileNotFound(t *testing.T) {
	s, _ := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/files/does-not-exist/download", nil)
	w := httptest.NewRecorder()

	s.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestDownloadFileSuccess(t *testing.T) {
	s, repo := newTestServer()
	ctx := context.Background()
	_ = repo.Create(ctx, &domain.File{
		ID: "abc123", Filename: "report.pdf", ContentType: "application/pdf",
		ObjectKey: "abc123/report.pdf", Status: domain.StatusProcessed,
	})

	req := httptest.NewRequest(http.MethodGet, "/files/abc123/download", nil)
	w := httptest.NewRecorder()
	s.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp downloadResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.DownloadURL == "" {
		t.Fatal("expected non-empty download_url")
	}
}
