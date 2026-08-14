// OCR document recognition (v2.0 F6). Default engine "tesseract_local" runs a
// self-hosted Tesseract container over HTTP — no third party receives the KTP/
// passport image (UU PDP data-minimization). "google_vision" remains as an
// optional fallback; "none" disables OCR while keeping everything compiling.
package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
)

// ErrOCRDisabled is returned when OCR is requested but no engine is configured.
var ErrOCRDisabled = errors.New("OCR engine tidak aktif")

// OCRExtraction is the structured result of OCR on a document (v2.0 F3).
type OCRExtraction struct {
	DocumentNumber string  `json:"document_number"` // NIK or passport number
	Name           string  `json:"name"`
	BirthDate      string  `json:"birth_date"`  // normalized YYYY-MM-DD when parseable
	Nationality    string  `json:"nationality"` // passport only
	ExpiryDate     string  `json:"expiry_date"` // passport only, YYYY-MM-DD
	Confidence     float64 `json:"confidence"`  // 0.0 - 1.0
}

// OCRValidation is the validation verdict for an extraction.
type OCRValidation struct {
	Passed bool
	Notes  string
}

// OCRService runs document OCR via a pluggable engine (v2.0 F6). "tesseract_local"
// talks HTTP to a self-hosted Tesseract sidecar (no cgo, no third party);
// "google_vision" is a non-cgo HTTP engine; "none" disables OCR.
type OCRService struct {
	engine       string
	visionKey    string
	tesseractURL string
	threshold    float64
	ocrRepo      document.OCRResultRepository
	parts        domainParticipant.Repository
	storage      *StorageService
	client       *http.Client
}

func NewOCRService(
	engine, visionKey, tesseractURL string,
	threshold float64,
	ocrRepo document.OCRResultRepository,
	parts domainParticipant.Repository,
	storage *StorageService,
) *OCRService {
	if threshold <= 0 {
		threshold = 0.85
	}
	if tesseractURL == "" {
		tesseractURL = "http://tesseract-api:8884"
	}
	return &OCRService{
		engine:       engine,
		visionKey:    visionKey,
		tesseractURL: tesseractURL,
		threshold:    threshold,
		ocrRepo:      ocrRepo,
		parts:        parts,
		storage:      storage,
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

// Enabled reports whether a real OCR engine is configured.
func (s *OCRService) Enabled() bool {
	return s != nil && s.engine != "" && s.engine != "none"
}

// PassportExpiry returns the OCR-extracted passport expiry (YYYY-MM-DD) for a
// participant and whether it expires within 6 months — or is already expired
// (v2.0 FR-PORTAL-11). Best-effort: returns ("", false) when unavailable.
func (s *OCRService) PassportExpiry(ctx context.Context, participantID string) (string, bool) {
	if s == nil || s.ocrRepo == nil {
		return "", false
	}
	expiry, err := s.ocrRepo.LatestPassportExpiry(ctx, participantID)
	if err != nil || expiry == "" {
		return "", false
	}
	exp, err := time.Parse("2006-01-02", expiry)
	if err != nil {
		return expiry, false
	}
	// "Expiring soon" = within 6 months from now (covers already-expired too).
	soon := time.Now().AddDate(0, 6, 0).After(exp)
	return expiry, soon
}

func isOCRDocType(t string) bool {
	t = strings.ToLower(t)
	return t == "ktp" || t == "passport" || t == "paspor"
}

// ProcessDocument runs OCR for a document and persists the result (async-safe).
// For KTP with high confidence it auto-fills participant.nik. Best-effort: logs
// and returns on any error.
func (s *OCRService) ProcessDocument(ctx context.Context, documentID, participantID, filePath, docType string) {
	if !s.Enabled() || !isOCRDocType(docType) {
		return
	}
	res, err := s.ExtractDocumentData(ctx, filePath, docType)
	if err != nil {
		log.Printf("OCR document %s failed: %v", documentID, err)
		return
	}

	var dep time.Time
	if p, err := s.parts.GetByID(ctx, participantID); err == nil && p.BatchDepartureDate != nil {
		dep = *p.BatchDepartureDate
	}
	val := s.ValidateOCRResult(res, docType, dep)

	data, _ := json.Marshal(res)
	if err := s.ocrRepo.Create(ctx, &document.OCRResult{
		DocumentID:       documentID,
		ExtractedData:    data,
		Confidence:       res.Confidence,
		ValidationPassed: val.Passed,
		ValidationNotes:  val.Notes,
	}); err != nil {
		log.Printf("OCR save result %s: %v", documentID, err)
		return
	}

	if strings.ToLower(docType) == "ktp" && res.Confidence >= s.threshold && res.DocumentNumber != "" {
		_ = s.parts.SetNIK(ctx, participantID, res.DocumentNumber)
	}
}

// docBucket is the private Supabase bucket where participant documents live.
const docBucket = "participant-documents"

// resolveURL turns a stored file_path into a fetchable URL. Document paths are
// stored as bucket-relative paths in a PRIVATE bucket, so they must be signed
// before download. A path that is already an absolute URL (manual-URL fallback
// when storage is disabled) is returned as-is.
func (s *OCRService) resolveURL(ctx context.Context, filePath string) (string, error) {
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		return filePath, nil
	}
	if s.storage == nil || !s.storage.Enabled() {
		return "", fmt.Errorf("storage tidak terkonfigurasi, tidak bisa resolve path privat %q", filePath)
	}
	return s.storage.SignedURL(ctx, docBucket, filePath, 600) // 10 menit cukup untuk OCR
}

// ExtractDocumentData downloads the file, recognizes its text with the engine,
// and parses it into structured fields.
func (s *OCRService) ExtractDocumentData(ctx context.Context, filePath, docType string) (*OCRExtraction, error) {
	url, err := s.resolveURL(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("resolve URL dokumen: %w", err)
	}
	img, err := s.fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("download dokumen: %w", err)
	}
	text, err := s.recognizeText(ctx, img)
	if err != nil {
		return nil, err
	}
	return ParseDocumentText(text, docType), nil
}

func (s *OCRService) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("download status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // cap 10MB
}

func (s *OCRService) recognizeText(ctx context.Context, img []byte) (string, error) {
	switch s.engine {
	case "tesseract_local", "tesseract":
		return s.tesseractLocalText(ctx, img)
	case "google_vision":
		return s.googleVisionText(ctx, img)
	default:
		return "", ErrOCRDisabled
	}
}

// tesseractLocalText recognizes text via a self-hosted Tesseract HTTP sidecar
// (e.g. hertzg/tesseract-server). The image never leaves the internal network.
func (s *OCRService) tesseractLocalText(ctx context.Context, img []byte) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	// hertzg/tesseract-server ships deu/eng/fra/kat/pol/rus/spa — no ind. Asking
	// for a language the image lacks fails the whole scan, it does not fall back.
	_ = w.WriteField("options", `{"languages":["eng"]}`)
	part, err := w.CreateFormFile("file", "document.jpg")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(img); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	url := strings.TrimRight(s.tesseractURL, "/") + "/tesseract"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tesseract sidecar: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("tesseract sidecar: status %d", resp.StatusCode)
	}
	var out struct {
		Data struct {
			Stdout string `json:"stdout"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("tesseract sidecar: decode: %w", err)
	}
	return out.Data.Stdout, nil
}

func (s *OCRService) googleVisionText(ctx context.Context, img []byte) (string, error) {
	payload := map[string]interface{}{
		"requests": []map[string]interface{}{{
			"image":    map[string]string{"content": base64.StdEncoding.EncodeToString(img)},
			"features": []map[string]string{{"type": "TEXT_DETECTION"}},
		}},
	}
	body, _ := json.Marshal(payload)
	url := "https://vision.googleapis.com/v1/images:annotate?key=" + s.visionKey
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Responses []struct {
			FullTextAnnotation struct {
				Text string `json:"text"`
			} `json:"fullTextAnnotation"`
		} `json:"responses"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Responses) == 0 {
		return "", fmt.Errorf("google vision: tidak ada teks terdeteksi")
	}
	return out.Responses[0].FullTextAnnotation.Text, nil
}

// ── Pure parsing + validation (engine-independent, unit-testable) ─────────────

var (
	reNIK      = regexp.MustCompile(`\b\d{16}\b`)
	reDate     = regexp.MustCompile(`\b(\d{2})[-/](\d{2})[-/](\d{4})\b`)
	rePassport = regexp.MustCompile(`\b([A-Z]\d{6,8})\b`)
	reMRZ      = regexp.MustCompile(`(?m)^[A-Z0-9<]{30,44}$`)
)

// ParseDocumentText extracts structured fields from raw OCR text. Confidence is
// derived from how many key fields were found, so it is meaningful even without
// engine-level confidence scores.
func ParseDocumentText(text, docType string) *OCRExtraction {
	if strings.EqualFold(docType, "ktp") {
		return parseKTP(text)
	}
	return parsePassport(text)
}

func parseKTP(text string) *OCRExtraction {
	r := &OCRExtraction{}
	conf := 0.0
	if m := reNIK.FindString(text); m != "" {
		r.DocumentNumber = m
		conf += 0.5
	}
	if name := findLabeled(text, "nama"); name != "" {
		r.Name = name
		conf += 0.3
	}
	if d := reDate.FindStringSubmatch(text); d != nil {
		r.BirthDate = d[3] + "-" + d[2] + "-" + d[1] // YYYY-MM-DD
		conf += 0.2
	}
	r.Confidence = conf
	return r
}

func parsePassport(text string) *OCRExtraction {
	r := &OCRExtraction{}
	conf := 0.0
	lines := reMRZ.FindAllString(strings.ToUpper(text), -1)
	if len(lines) >= 2 {
		// TD3 passport MRZ: line2 = passportNo(9) check nat(3) DOB(6) ... expiry(6)
		l1, l2 := lines[len(lines)-2], lines[len(lines)-1]
		if len(l2) >= 28 {
			r.DocumentNumber = strings.TrimRight(strings.ReplaceAll(l2[0:9], "<", ""), "<")
			r.Nationality = l2[10:13]
			r.BirthDate = mrzDate(l2[13:19], false)
			r.ExpiryDate = mrzDate(l2[21:27], true)
			if r.DocumentNumber != "" {
				conf += 0.4
			}
			if r.ExpiryDate != "" {
				conf += 0.15
			}
			if r.BirthDate != "" {
				conf += 0.15
			}
		}
		if name := mrzName(l1); name != "" {
			r.Name = name
			conf += 0.3
		}
	} else {
		// Fallback: regex-based extraction from plain text.
		if m := rePassport.FindStringSubmatch(text); m != nil {
			r.DocumentNumber = m[1]
			conf += 0.3
		}
		if name := findLabeled(text, "name"); name != "" {
			r.Name = name
			conf += 0.2
		}
	}
	r.Confidence = conf
	return r
}

// findLabeled returns the text after a label like "Nama : X" / "NAMA X".
//
// Match and cut are made on the same string. They used to differ: the match was
// against the trimmed line, the cut was `line[len(label):]` on the untrimmed
// one, so a scan that put two spaces before "Nama" ate the first two characters
// of the name instead — and OCR output is full of leading whitespace, which is
// why almost every scanned name needed correcting by hand.
func findLabeled(text, label string) string {
	label = strings.ToLower(label)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(line), label) {
			continue
		}
		rest := strings.TrimLeft(line[len(label):], ": \t")
		if rest = strings.TrimSpace(rest); rest != "" {
			return rest
		}
	}
	return ""
}

// mrzName parses the name from MRZ line 1: P<IDNSURNAME<<GIVEN<NAMES<<<.
func mrzName(l1 string) string {
	i := strings.Index(l1, "<")
	if i < 0 || len(l1) < 5 {
		return ""
	}
	body := l1[5:] // after "P<IDN"
	body = strings.ReplaceAll(body, "<<", " ")
	body = strings.ReplaceAll(body, "<", " ")
	return strings.TrimSpace(body)
}

// mrzDate converts a YYMMDD MRZ date to YYYY-MM-DD. `future` biases century to 20xx.
func mrzDate(s string, future bool) string {
	if len(s) != 6 || strings.Contains(s, "<") {
		return ""
	}
	yy, mm, dd := s[0:2], s[2:4], s[4:6]
	century := "19"
	if future || yy < "50" {
		century = "20"
	}
	return century + yy + "-" + mm + "-" + dd
}

// ValidateOCRResult validates an extraction against business rules.
func (s *OCRService) ValidateOCRResult(r *OCRExtraction, docType string, departureDate time.Time) OCRValidation {
	if r.Confidence < 0.5 {
		return OCRValidation{Passed: false, Notes: "Kualitas gambar kurang baik, perlu review manual"}
	}
	switch strings.ToLower(docType) {
	case "ktp":
		if len(r.DocumentNumber) != 16 || !isAllDigits(r.DocumentNumber) {
			return OCRValidation{Passed: false, Notes: "NIK tidak valid (harus 16 digit angka)"}
		}
		return OCRValidation{Passed: true, Notes: "NIK valid"}
	case "passport", "paspor":
		exp, err := time.Parse("2006-01-02", r.ExpiryDate)
		if err != nil {
			return OCRValidation{Passed: false, Notes: "Tanggal berlaku paspor tidak terbaca"}
		}
		if !departureDate.IsZero() && exp.Before(departureDate.AddDate(0, 6, 0)) {
			return OCRValidation{Passed: false, Notes: "Paspor harus berlaku minimal 6 bulan setelah tanggal keberangkatan"}
		}
		return OCRValidation{Passed: true, Notes: "Paspor berlaku > 6 bulan dari tanggal keberangkatan"}
	}
	return OCRValidation{Passed: false, Notes: "Jenis dokumen tidak didukung OCR"}
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
