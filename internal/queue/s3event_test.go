package queue

import (
	"errors"
	"testing"
)

const validEvent = `{
  "Records": [
    {
      "eventName": "ObjectCreated:Put",
      "s3": {
        "bucket": {"name": "securelink-files"},
        "object": {"key": "abc123/report.pdf", "size": 1024}
      }
    }
  ]
}`

func TestParseS3EventValid(t *testing.T) {
	events, err := ParseS3Event(validEvent)
	if err != nil {
		t.Fatalf("ParseS3Event: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	e := events[0]
	if e.FileID != "abc123" || e.Key != "abc123/report.pdf" || e.Bucket != "securelink-files" || e.Size != 1024 {
		t.Fatalf("unexpected event: %+v", e)
	}
}

func TestParseS3EventMalformed(t *testing.T) {
	cases := []string{
		"not json at all",
		"{}",
		`{"Records": []}`,
		`{"Records": [{"eventName": "x", "s3": {"bucket": {"name": ""}, "object": {"key": "abc/x"}}}]}`,
		`{"Records": [{"eventName": "x", "s3": {"bucket": {"name": "b"}, "object": {"key": "no-slash-key"}}}]}`,
	}
	for _, body := range cases {
		_, err := ParseS3Event(body)
		if !errors.Is(err, ErrMalformedEvent) {
			t.Errorf("ParseS3Event(%q) error = %v, want ErrMalformedEvent", body, err)
		}
	}
}
