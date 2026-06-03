package service

import "testing"

func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"Paspor Budi.pdf":      "Paspor-Budi.pdf",
		"hello@world.txt":      "helloworld.txt",
		"foto-paspor_v2.jpg":   "foto-paspor_v2.jpg",
		"file with spaces":     "file-with-spaces",
		"":                     "file",
		"!!!#$%":               "file",
		"normal-file.pdf":      "normal-file.pdf",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsPublicBucket(t *testing.T) {
	cases := map[string]bool{
		"package-images":         true,
		"tour-leader-photos":     true,
		"participant-documents":  false,
		"invoices-pdf":           false,
		"payment-proofs":         false,
	}
	for bucket, want := range cases {
		if got := isPublicBucket(bucket); got != want {
			t.Errorf("isPublicBucket(%q) = %v, want %v", bucket, got, want)
		}
	}
}

func TestDetectContentType(t *testing.T) {
	cases := map[string]string{
		".jpg":  "image/jpeg",
		".JPG":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".pdf":  "application/pdf",
		".webp": "image/webp",
		".xyz":  "application/octet-stream",
	}
	for ext, want := range cases {
		if got := detectContentType(ext); got != want {
			t.Errorf("detectContentType(%q) = %q, want %q", ext, got, want)
		}
	}
}

func TestExtractJSONField(t *testing.T) {
	body := `{"signedURL": "/object/sign/bucket/path?token=abc", "other": "value"}`
	if got := extractJSONField(body, "signedURL"); got != "/object/sign/bucket/path?token=abc" {
		t.Errorf("got %q", got)
	}
	if got := extractJSONField(body, "other"); got != "value" {
		t.Errorf("got %q for 'other'", got)
	}
	if got := extractJSONField(body, "missing"); got != "" {
		t.Errorf("expected empty for missing field, got %q", got)
	}
}

func TestStorageServiceEnabled(t *testing.T) {
	disabled := NewStorageService("", "")
	if disabled.Enabled() {
		t.Error("expected disabled when no creds")
	}
	enabled := NewStorageService("https://x.supabase.co", "secret")
	if !enabled.Enabled() {
		t.Error("expected enabled with creds")
	}
}
