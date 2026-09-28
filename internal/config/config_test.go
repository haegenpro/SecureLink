package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()
	if cfg.APIAddr != ":8080" {
		t.Errorf("APIAddr = %q, want :8080", cfg.APIAddr)
	}
	if cfg.UploadURLTTL.Seconds() != 300 {
		t.Errorf("UploadURLTTL = %v, want 300s", cfg.UploadURLTTL)
	}
	if cfg.SQSMaxMessages != 5 {
		t.Errorf("SQSMaxMessages = %d, want 5", cfg.SQSMaxMessages)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("API_ADDR", ":9090")
	t.Setenv("UPLOAD_URL_TTL_SECONDS", "60")
	cfg := Load()
	if cfg.APIAddr != ":9090" {
		t.Errorf("APIAddr = %q, want :9090", cfg.APIAddr)
	}
	if cfg.UploadURLTTL.Seconds() != 60 {
		t.Errorf("UploadURLTTL = %v, want 60s", cfg.UploadURLTTL)
	}
}
