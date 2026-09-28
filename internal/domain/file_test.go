package domain

import "testing"

func TestValidateFilename(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"empty", "", true},
		{"valid", "report.pdf", false},
		{"path separator unix", "../etc/passwd", true},
		{"path separator windows", "..\\..\\windows\\system32", true},
		{"dot", ".", true},
		{"dotdot", "..", true},
		{"nested path", "a/b.txt", true},
		{"too long", string(make([]byte, 300)), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFilename(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateFilename(%q) error = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestValidateContentType(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"empty", "", true},
		{"valid", "application/pdf", false},
		{"no subtype", "application/", true},
		{"no slash", "application", true},
		{"control char", "application/pdf\n", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateContentType(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateContentType(%q) error = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestSanitizedObjectKey(t *testing.T) {
	got := SanitizedObjectKey("abc123", "report.pdf")
	want := "abc123/report.pdf"
	if got != want {
		t.Fatalf("SanitizedObjectKey = %q, want %q", got, want)
	}
}

func TestStatusValid(t *testing.T) {
	for _, s := range []Status{StatusPending, StatusProcessing, StatusProcessed, StatusFailed} {
		if !s.Valid() {
			t.Fatalf("expected %q to be valid", s)
		}
	}
	if Status("bogus").Valid() {
		t.Fatal("expected bogus status to be invalid")
	}
}
