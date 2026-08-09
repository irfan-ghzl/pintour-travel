package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
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
	Bucket    string
	Path      string // bucket-relative path
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

	// The extension is sanitised as well as the stem. It used to be taken from
	// the uploaded name verbatim and appended after the cleaning, so everything
	// sanitizeFilename refused could be smuggled back in behind the last dot —
	// including the slashes and dots that reach a different object entirely.
	rawExt := filepath.Ext(fileHeader.Filename)
	ext := sanitizeExtension(rawExt)
	cleanName := sanitizeFilename(strings.TrimSuffix(fileHeader.Filename, rawExt))
	objectPath := fmt.Sprintf("%s/%d-%s%s", ownerID, time.Now().UnixNano(), cleanName, ext)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.objectURL("object", bucket, objectPath), file)
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
		publicURL = s.objectURL("object/public", bucket, objectPath)
	}
	return &UploadResult{Bucket: bucket, Path: objectPath, PublicURL: publicURL}, nil
}

// SignedURL generates a time-limited URL for accessing a private object (§19.2).
// Expiry: 3600s (1 hour) per PRD §19.2.
func (s *StorageService) SignedURL(ctx context.Context, bucket, objectPath string, expirySec int) (string, error) {
	if !s.enabled {
		return "", fmt.Errorf("storage not configured")
	}
	body := fmt.Sprintf(`{"expiresIn": %d}`, expirySec)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.objectURL("object/sign", bucket, objectPath), bytes.NewBufferString(body))
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
	// Response: {"signedURL": "/object/sign/bucket/path?token=..."}. Supabase has
	// shipped both spellings of the key, so both are accepted.
	var signed struct {
		SignedURLUpper string `json:"signedURL"`
		SignedURLMixed string `json:"signedUrl"`
	}
	if err := json.Unmarshal(respBody, &signed); err != nil {
		return "", fmt.Errorf("decode sign response: %w", err)
	}
	signedPath := signed.SignedURLUpper
	if signedPath == "" {
		signedPath = signed.SignedURLMixed
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
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		s.objectURL("object", bucket, objectPath), nil)
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

// objectURL builds a storage endpoint with every path component escaped.
//
// The components used to be pasted into the URL as-is. A bucket or object name
// carrying a "?", "#" or "../" then changed which request was being made rather
// than which object it named — and object names come from filenames a
// participant chose.
func (s *StorageService) objectURL(endpoint, bucket, objectPath string) string {
	return fmt.Sprintf("%s/storage/v1/%s/%s/%s",
		s.baseURL, endpoint, url.PathEscape(bucket), escapeObjectPath(objectPath))
}

// escapeObjectPath escapes each segment of a bucket-relative path, keeping the
// separators between them — a stored path is several segments, and encoding its
// slashes would name a single object whose name contains them.
func escapeObjectPath(objectPath string) string {
	segments := strings.Split(objectPath, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}

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

// sanitizeExtension reduces a file extension to a leading dot and letters and
// digits, which is all any extension this service accepts consists of. An empty
// or unrecognisable extension yields none at all rather than a guess.
func sanitizeExtension(ext string) string {
	if !strings.HasPrefix(ext, ".") {
		return ""
	}
	var b strings.Builder
	b.WriteByte('.')
	for _, r := range ext[1:] {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	if b.Len() == 1 {
		return ""
	}
	return b.String()
}
