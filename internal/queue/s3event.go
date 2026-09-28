package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ErrMalformedEvent is returned when a message body is not a recognizable S3
// event notification.
var ErrMalformedEvent = errors.New("malformed S3 event")

// s3Notification mirrors the subset of the S3 -> SQS event notification
// payload this project cares about.
type s3Notification struct {
	Records []struct {
		EventName string `json:"eventName"`
		S3        struct {
			Bucket struct {
				Name string `json:"name"`
			} `json:"bucket"`
			Object struct {
				Key  string `json:"key"`
				Size int64  `json:"size"`
			} `json:"object"`
		} `json:"s3"`
	} `json:"Records"`
}

// ObjectEvent is one parsed S3 object-created record, with the object key
// decoded and the file ID (its first path segment) extracted.
type ObjectEvent struct {
	EventName string
	Bucket    string
	Key       string
	FileID    string
	Size      int64
}

// ParseS3Event decodes an SQS message body as an S3 event notification and
// returns one ObjectEvent per record. It returns ErrMalformedEvent (wrapped
// with detail) for any body that isn't a valid, well-formed S3 notification,
// so the worker can treat it as a permanent, non-retryable failure rather
// than crash.
func ParseS3Event(body string) ([]ObjectEvent, error) {
	var n s3Notification
	if err := json.Unmarshal([]byte(body), &n); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedEvent, err)
	}
	if len(n.Records) == 0 {
		return nil, fmt.Errorf("%w: no records", ErrMalformedEvent)
	}

	events := make([]ObjectEvent, 0, len(n.Records))
	for _, rec := range n.Records {
		if rec.S3.Bucket.Name == "" || rec.S3.Object.Key == "" {
			return nil, fmt.Errorf("%w: missing bucket or key", ErrMalformedEvent)
		}
		// S3 URL-encodes object keys in event notifications.
		key, err := url.QueryUnescape(rec.S3.Object.Key)
		if err != nil {
			return nil, fmt.Errorf("%w: undecodable key: %v", ErrMalformedEvent, err)
		}
		fileID, ok := fileIDFromKey(key)
		if !ok {
			return nil, fmt.Errorf("%w: key %q has no file ID prefix", ErrMalformedEvent, key)
		}
		events = append(events, ObjectEvent{
			EventName: rec.EventName,
			Bucket:    rec.S3.Bucket.Name,
			Key:       key,
			FileID:    fileID,
			Size:      rec.S3.Object.Size,
		})
	}
	return events, nil
}

// fileIDFromKey extracts the file ID from an object key of the form
// "<file_id>/<filename>", matching storage.Uploader's key layout.
func fileIDFromKey(key string) (string, bool) {
	idx := strings.Index(key, "/")
	if idx <= 0 || idx == len(key)-1 {
		return "", false
	}
	return key[:idx], true
}
