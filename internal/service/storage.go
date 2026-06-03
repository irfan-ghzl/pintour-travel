package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// StorageService integrates with Supabase Storage per PRD §16.2.
//
// Buckets used (per PRD):
//   - package-images (public)
//   - participant-documents (private, signed URL §19.2)
//   - invoices-pdf (private)
//   - tour-leader-photos (public)
type StorageService struct {
	baseURL    string // https://xxx.supabase.co
	serviceKey string // service_role key
	client     *http.Client
	// Local fallback used when serviceKey is empty (for development without Supabase).
	enabled bool
}

func NewStorageService(supabaseURL, serviceKey string) *StorageService {
	return &StorageService{
		baseURL:    strings.TrimRight(supabaseURL, "/"),
		serviceKey: serviceKey,
		client:     &http.Client{Timeout: 30 * time.Second},
		enabled:    supabaseURL != "" && serviceKey != "",
	}
}

// Enabled reports whether storage is configured.
func (s *StorageService) Enabled() bool { return s.enabled }

// UploadResult is returned after a successful upload.
type UploadResult struct {
	Bucket   string
	Path     string // bucket-relative path
	PublicURL string // for public buckets
}

// Upload uploads a file to the given bucket and returns the stored path.
// File path inside bucket: {participantID or packageID}/{timestamp}-{filename}
func (s *StorageService) Upload(ctx context.Context, bucket, ownerID string, fileHeader *multipart.FileHeader) (*UploadResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("storage service not configured (set SUPABASE_URL and SUPABASE_SERVICE_KEY)")
	}
	if fileHeader.Size > 5*1024*1024 {
		return nil, fmt.Errorf("ukuran file melebihi 5MB")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ext := filepath.Ext(fileHeader.Filename)
	cleanName := sanitizeFilename(strings.TrimSuffix(fileHeader.Filename, ext))
	objectPath := fmt.Sprintf("%s/%d-%s%s", ownerID, time.Now().UnixNano(), cleanName, ext)

	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, bucket, objectPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, file)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Content-Type", detectContentType(ext))
	req.Header.Set("x-upsert", "false")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase upload failed: %s — %s", resp.Status, string(body))
	}

	publicURL := ""
	if isPublicBucket(bucket) {
		publicURL = fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.baseURL, bucket, objectPath)
	}
	return &UploadResult{Bucket: bucket, Path: objectPath, PublicURL: publicURL}, nil
}

// SignedURL generates a time-limited URL for accessing a private object (§19.2).
// Expiry: 3600s (1 hour) per PRD §19.2.
func (s *StorageService) SignedURL(ctx context.Context, bucket, objectPath string, expirySec int) (string, error) {
	if !s.enabled {
		return "", fmt.Errorf("storage not configured")
	}
	url := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s", s.baseURL, bucket, objectPath)
	body := fmt.Sprintf(`{"expiresIn": %d}`, expirySec)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("supabase sign failed: %s", resp.Status)
	}
	respBody, _ := io.ReadAll(resp.Body)
	// Response: {"signedURL": "/object/sign/bucket/path?token=..."}
	signedPath := extractJSONField(string(respBody), "signedURL")
	if signedPath == "" {
		signedPath = extractJSONField(string(respBody), "signedUrl")
	}
	if signedPath == "" {
		return "", fmt.Errorf("no signed URL in response")
	}
	return s.baseURL + "/storage/v1" + signedPath, nil
}

// Delete removes an object from a bucket.
func (s *StorageService) Delete(ctx context.Context, bucket, objectPath string) error {
	if !s.enabled {
		return fmt.Errorf("storage not configured")
	}
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, bucket, objectPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("supabase delete failed: %s", resp.Status)
	}
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func isPublicBucket(bucket string) bool {
	return bucket == "package-images" || bucket == "tour-leader-photos"
}

func detectContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".pdf":
		return "application/pdf"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func sanitizeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	out := b.String()
	if out == "" {
		out = "file"
	}
	return out
}

// extractJSONField does a minimal extraction of a single string field
// to avoid pulling in encoding/json for this micro-task.
func extractJSONField(body, field string) string {
	key := `"` + field + `":`
	idx := strings.Index(body, key)
	if idx == -1 {
		return ""
	}
	rest := strings.TrimSpace(body[idx+len(key):])
	if !strings.HasPrefix(rest, `"`) {
		return ""
	}
	rest = rest[1:]
	end := strings.Index(rest, `"`)
	if end == -1 {
		return ""
	}
	return rest[:end]
}

// ComputeETag returns a simple HMAC tag (not used directly by Supabase but
// available for integrity verification).
func (s *StorageService) ComputeETag(content []byte) string {
	h := hmac.New(sha256.New, []byte(s.serviceKey))
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}
