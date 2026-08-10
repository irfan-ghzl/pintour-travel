package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"Paspor Budi.pdf":    "Paspor-Budi.pdf",
		"hello@world.txt":    "helloworld.txt",
		"foto-paspor_v2.jpg": "foto-paspor_v2.jpg",
		"file with spaces":   "file-with-spaces",
		"":                   "file",
		"!!!#$%":             "file",
		"normal-file.pdf":    "normal-file.pdf",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsPublicBucket(t *testing.T) {
	cases := map[string]bool{
		"package-images":        true,
		"tour-leader-photos":    true,
		"participant-documents": false,
		"invoices-pdf":          false,
		"payment-proofs":        false,
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

// The signed URL is read with encoding/json now, so a response that quotes a
// brace or escapes a character no longer confuses the reader. Both spellings
// Supabase has shipped for the key are still accepted.
func TestSignedURLReadsBothKeySpellings(t *testing.T) {
	cases := map[string]string{
		`{"signedURL":"/object/sign/bucket/a?token=abc"}`:      "/object/sign/bucket/a?token=abc",
		`{"signedUrl":"/object/sign/bucket/a?token=abc"}`:      "/object/sign/bucket/a?token=abc",
		`{"other":"x","signedURL":"/object/sign/b/c?token=d"}`: "/object/sign/b/c?token=d",
		`{"signedURL":"/object/sign/b/na\"me?token=d"}`:        `/object/sign/b/na"me?token=d`,
		`{"error":"not found"}`:                                "",
	}
	for body, want := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		}))
		s := NewStorageService(srv.URL, "key")
		got, err := s.SignedURL(context.Background(), "participant-documents", "p-1/passport.pdf", 3600)
		srv.Close()
		switch {
		case want == "":
			if err == nil {
				t.Errorf("body %s: err = nil, want error", body)
			}
		case err != nil:
			t.Errorf("body %s: %v", body, err)
		case got != srv.URL+"/storage/v1"+want:
			t.Errorf("body %s: got %q, want suffix %q", body, got, want)
		}
	}
}

// An extension is cleaned the same way the rest of the name is. It used to be
// appended verbatim, so anything sanitizeFilename refused could be smuggled
// back in behind the last dot.
func TestSanitizeExtension(t *testing.T) {
	cases := map[string]string{
		".pdf":         ".pdf",
		".JPG":         ".JPG",
		".jpeg":        ".jpeg",
		"":             "",
		"pdf":          "", // no leading dot is not an extension
		".p df":        ".pdf",
		"./../etc":     ".etc", // path syntax stripped, letters kept
		".pdf/../../x": ".pdfx",
		".pdf?raw=1":   ".pdfraw1",
		".":            "",
		".!!!":         "",
	}
	for in, want := range cases {
		if got := sanitizeExtension(in); got != want {
			t.Errorf("sanitizeExtension(%q) = %q, want %q", in, got, want)
		}
	}
}

// Every path component is escaped when the request URL is built, so a name that
// carries URL syntax names an object rather than changing the request.
func TestObjectURLEscapesComponents(t *testing.T) {
	s := NewStorageService("https://x.supabase.co", "key")
	cases := map[string]string{
		"p-1/passport.pdf":     "https://x.supabase.co/storage/v1/object/b/p-1/passport.pdf",
		"p-1/na me.pdf":        "https://x.supabase.co/storage/v1/object/b/p-1/na%20me.pdf",
		"p-1/x?token=leak.pdf": "https://x.supabase.co/storage/v1/object/b/p-1/x%3Ftoken=leak.pdf",
		"p-1/x#frag.pdf":       "https://x.supabase.co/storage/v1/object/b/p-1/x%23frag.pdf",
	}
	for objectPath, want := range cases {
		if got := s.objectURL("object", "b", objectPath); got != want {
			t.Errorf("objectURL(%q) = %q, want %q", objectPath, got, want)
		}
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
