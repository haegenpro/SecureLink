package api

import "time"

type healthResponse struct {
	Status string `json:"status"`
}

type createFileRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type createFileResponse struct {
	FileID    string `json:"file_id"`
	UploadURL string `json:"upload_url"`
	ExpiresIn int    `json:"expires_in"`
}

type fileStatusResponse struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}

type downloadResponse struct {
	DownloadURL string `json:"download_url"`
	ExpiresIn   int    `json:"expires_in"`
}

type errorResponse struct {
	Error string `json:"error"`
}
