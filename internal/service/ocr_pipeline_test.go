package service

// Ticket 13 — the OCR pipeline (v2.0 F6) and the storage it reads through,
// against stand-ins for both.
//
// The promise to an admin is that a scanned KTP fills in the NIK by itself, and
// that a scan it could not read leaves the record alone rather than writing a
// guess. Both are asserted end to end here: the document is signed for, fetched,
// recognised, parsed, validated, stored, and — only when it is a confident KTP —
// applied to the participant.

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
)

// ─── Fakes ────────────────────────────────────────────────────────────────────

type fakeOCRStore struct {
	mu      sync.Mutex
	results []document.OCRResult
}

func (r *fakeOCRStore) Create(_ context.Context, res *document.OCRResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results = append(r.results, *res)
	return nil
}

func (r *fakeOCRStore) GetByDocument(_ context.Context, documentID string) (*document.OCRResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.results {
		if r.results[i].DocumentID == documentID {
			return &r.results[i], nil
		}
	}
	return nil, nil
}

func (r *fakeOCRStore) LatestPassportExpiry(context.Context, string) (string, error) {
	return "", nil
}

func (r *fakeOCRStore) last(t *testing.T) document.OCRResult {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.results) == 0 {
		t.Fatal("tidak ada hasil OCR yang tersimpan")
	}
	return r.results[len(r.results)-1]
}

// participantStore records only what the OCR path needs: reading a participant
// and writing back a NIK.
type participantStore struct {
	mu        sync.Mutex
	departure *time.Time
	nik       string
}

func (r *participantStore) Create(context.Context, *domainParticipant.Participant) error { return nil }
func (r *participantStore) Update(context.Context, *domainParticipant.Participant) error { return nil }
func (r *participantStore) GetByID(_ context.Context, id string) (*domainParticipant.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return &domainParticipant.Participant{ID: id, BatchDepartureDate: r.departure}, nil
}
func (r *participantStore) GetByPhone(context.Context, string) (*domainParticipant.Participant, error) {
	return nil, nil
}
func (r *participantStore) ListByPortalUser(context.Context, string, string) ([]domainParticipant.Participant, error) {
	return nil, nil
}
func (r *participantStore) List(context.Context, domainParticipant.Filter) ([]domainParticipant.Participant, int, error) {
	return nil, 0, nil
}
func (r *participantStore) Activate(context.Context, string) error { return nil }
func (r *participantStore) SetNIK(_ context.Context, _, nik string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nik = nik
	return nil
}
func (r *participantStore) ListByBatch(context.Context, string) ([]domainParticipant.Participant, error) {
	return nil, nil
}
func (r *participantStore) ListByDepartureDaysAhead(context.Context, int) ([]domainParticipant.Participant, error) {
	return nil, nil
}
func (r *participantStore) storedNIK() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.nik
}

// ─── Harness ──────────────────────────────────────────────────────────────────

// newOCR wires an OCR service onto stand-ins for storage and the recognition
// sidecar. recognised is the text the sidecar "reads" off the image.
func newOCR(t *testing.T, recognised string) (*OCRService, *fakeOCRStore, *participantStore) {
	t.Helper()

	// The storage stand-in serves both roles: it signs a private path, and it
	// serves the object the signature points at.
	var storageURL string
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/object/sign/") {
			_ = json.NewEncoder(w).Encode(map[string]string{
				"signedURL": "/object/download/participant-documents/p1/ktp.jpg?token=abc",
			})
			return
		}
		_, _ = w.Write([]byte("gambar-palsu"))
	}))
	t.Cleanup(storage.Close)
	storageURL = storage.URL

	sidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The image is posted as multipart, the way the sidecar expects it.
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"stdout": recognised},
		})
	}))
	t.Cleanup(sidecar.Close)

	ocrRepo := &fakeOCRStore{}
	parts := &participantStore{}
	svc := NewOCRService("tesseract_local", "", sidecar.URL, 0.85, ocrRepo, parts,
		NewStorageService(storageURL, "service-key"))
	return svc, ocrRepo, parts
}

const ktpScan = `PROVINSI DKI JAKARTA
  NIK : 3174012501900001
  Nama : JOHN DOE
  Tempat/Tgl Lahir : JAKARTA, 25-01-1990`

// ─── Tests ────────────────────────────────────────────────────────────────────

// A KTP the engine read confidently fills in the participant's NIK by itself,
// and the extraction is stored so an admin can check what was read.
func TestOCR_ConfidentKTPFillsInTheNIK(t *testing.T) {
	svc, results, parts := newOCR(t, ktpScan)

	svc.ProcessDocument(context.Background(), "doc-1", "participant-1",
		"participant-documents/p1/ktp.jpg", "ktp")

	if got := parts.storedNIK(); got != "3174012501900001" {
		t.Errorf("NIK terisi = %q, want 3174012501900001", got)
	}
	res := results.last(t)
	if res.DocumentID != "doc-1" {
		t.Errorf("hasil tersimpan untuk dokumen %q", res.DocumentID)
	}
	if !res.ValidationPassed {
		t.Errorf("validasi gagal: %s", res.ValidationNotes)
	}
	var extracted OCRExtraction
	if err := json.Unmarshal(res.ExtractedData, &extracted); err != nil {
		t.Fatalf("hasil ekstraksi tidak terbaca: %v", err)
	}
	if extracted.Name != "JOHN DOE" {
		t.Errorf("nama terbaca = %q, want JOHN DOE", extracted.Name)
	}
}

// A scan the engine could not make sense of is recorded as failed validation
// and leaves the participant's NIK alone. Writing a guess would be worse than
// writing nothing: nobody would know to check it.
func TestOCR_UnreadableScanWritesNoNIK(t *testing.T) {
	svc, results, parts := newOCR(t, "gambar buram tidak terbaca")

	svc.ProcessDocument(context.Background(), "doc-1", "participant-1",
		"participant-documents/p1/ktp.jpg", "ktp")

	if got := parts.storedNIK(); got != "" {
		t.Errorf("NIK terisi %q dari pindaian yang tidak terbaca", got)
	}
	if res := results.last(t); res.ValidationPassed {
		t.Error("validasi lolos untuk pindaian yang tidak terbaca")
	}
}

// A passport is read and stored, but never writes a NIK — that field belongs to
// the KTP.
func TestOCR_PassportNeverWritesANIK(t *testing.T) {
	mrz := "P<IDNDOE<<JOHN<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<\n" +
		"A1234567<0IDN9001010M3001010<<<<<<<<<<<<<<00"
	svc, results, parts := newOCR(t, mrz)

	svc.ProcessDocument(context.Background(), "doc-1", "participant-1",
		"participant-documents/p1/passport.jpg", "passport")

	if got := parts.storedNIK(); got != "" {
		t.Errorf("NIK terisi %q dari paspor", got)
	}
	var extracted OCRExtraction
	_ = json.Unmarshal(results.last(t).ExtractedData, &extracted)
	if extracted.DocumentNumber != "A1234567" {
		t.Errorf("nomor paspor = %q, want A1234567", extracted.DocumentNumber)
	}
}

// Only the document types OCR is meant for are processed. Running it over a
// bank statement would store a meaningless extraction against it.
func TestOCR_SkipsDocumentTypesItCannotRead(t *testing.T) {
	for _, docType := range []string{"rekening_koran", "visa_support", "lainnya", ""} {
		t.Run(docType, func(t *testing.T) {
			svc, results, _ := newOCR(t, ktpScan)
			svc.ProcessDocument(context.Background(), "doc-1", "participant-1",
				"participant-documents/p1/berkas.pdf", docType)
			if len(results.results) != 0 {
				t.Errorf("hasil tersimpan = %d untuk tipe %q, want 0", len(results.results), docType)
			}
		})
	}
}

func TestIsOCRDocType(t *testing.T) {
	cases := map[string]bool{
		"ktp": true, "KTP": true, "passport": true, "paspor": true, "Passport": true,
		"rekening_koran": false, "visa_support": false, "lainnya": false, "": false,
	}
	for in, want := range cases {
		if got := isOCRDocType(in); got != want {
			t.Errorf("isOCRDocType(%q) = %v, want %v", in, got, want)
		}
	}
}

// A private path cannot be fetched without storage to sign it, and the error
// says so rather than failing later with a confusing download error.
func TestOCR_PrivatePathNeedsStorage(t *testing.T) {
	svc := NewOCRService("tesseract_local", "", "http://unused", 0.85,
		&fakeOCRStore{}, &participantStore{}, NewStorageService("", ""))

	_, err := svc.ExtractDocumentData(context.Background(),
		"participant-documents/p1/ktp.jpg", "ktp")
	if err == nil {
		t.Fatal("path privat tanpa penyimpanan diterima")
	}
	if !strings.Contains(err.Error(), "storage") {
		t.Errorf("pesan %q tidak menyebut penyimpanan", err)
	}
}

// An engine nobody configured recognises nothing, and says so.
func TestOCR_DisabledEngineIsRefused(t *testing.T) {
	svc := NewOCRService("", "", "", 0.85, &fakeOCRStore{}, &participantStore{}, nil)
	if svc.Enabled() {
		t.Error("Enabled() = true tanpa mesin OCR")
	}
	svc.ProcessDocument(context.Background(), "doc-1", "participant-1", "path", "ktp")
}

// ─── Storage upload / delete ──────────────────────────────────────────────────

// An uploaded document is stored under the owner's folder with a name derived
// from — but not equal to — the one the participant chose.
func TestStorageUpload_StoresUnderTheOwnerWithASafeName(t *testing.T) {
	var gotPath, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := NewStorageService(srv.URL, "service-key")
	res, err := svc.Upload(context.Background(), "participant-documents", "participant-1",
		fileHeaderFor(t, "Paspor Budi.pdf", []byte("%PDF-1.4")))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if !strings.HasPrefix(res.Path, "participant-1/") {
		t.Errorf("path = %q, want di bawah folder pemiliknya", res.Path)
	}
	if !strings.HasSuffix(res.Path, "-Paspor-Budi.pdf") {
		t.Errorf("path = %q, want nama tersanitasi dengan ekstensi utuh", res.Path)
	}
	if gotContentType != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", gotContentType)
	}
	if !strings.Contains(gotPath, "participant-documents") {
		t.Errorf("URL permintaan = %q, tidak menyebut bucketnya", gotPath)
	}
	// A private bucket gets no public URL — that is the whole point of it.
	if res.PublicURL != "" {
		t.Errorf("PublicURL = %q untuk bucket privat", res.PublicURL)
	}
}

// A public bucket does hand back a URL anyone can open, which is what the
// catalogue's images are for.
func TestStorageUpload_PublicBucketReturnsAPublicURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := NewStorageService(srv.URL, "service-key")
	res, err := svc.Upload(context.Background(), "package-images", "package-1",
		fileHeaderFor(t, "masjidil-haram.jpg", []byte("\xff\xd8\xff")))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if !strings.Contains(res.PublicURL, "/object/public/package-images/") {
		t.Errorf("PublicURL = %q", res.PublicURL)
	}
}

// A file over the §16.2 ceiling is refused before it is sent anywhere.
func TestStorageUpload_RefusesOversizedFiles(t *testing.T) {
	var reached bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := NewStorageService(srv.URL, "service-key")
	header := fileHeaderFor(t, "besar.pdf", []byte("kecil"))
	header.Size = 6 * 1024 * 1024

	if _, err := svc.Upload(context.Background(), "participant-documents", "p-1", header); err == nil {
		t.Fatal("berkas 6MB diterima, want ditolak")
	}
	if reached {
		t.Error("berkas terlanjur dikirim ke penyimpanan sebelum ukurannya ditolak")
	}
}

// Without credentials nothing is uploaded, signed or deleted, and each says so.
func TestStorage_UnconfiguredRefusesEveryOperation(t *testing.T) {
	svc := NewStorageService("", "")
	ctx := context.Background()

	if _, err := svc.Upload(ctx, "participant-documents", "p-1",
		fileHeaderFor(t, "a.pdf", []byte("x"))); err == nil {
		t.Error("Upload tanpa konfigurasi diterima")
	}
	if _, err := svc.SignedURL(ctx, "participant-documents", "p-1/a.pdf", 3600); err == nil {
		t.Error("SignedURL tanpa konfigurasi diterima")
	}
	if err := svc.Delete(ctx, "participant-documents", "p-1/a.pdf"); err == nil {
		t.Error("Delete tanpa konfigurasi diterima")
	}
}

// Deleting names the object being removed, and an upstream refusal is reported.
func TestStorageDelete(t *testing.T) {
	var deleted string
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleted = r.URL.Path
		w.WriteHeader(status)
	}))
	defer srv.Close()

	svc := NewStorageService(srv.URL, "service-key")
	if err := svc.Delete(context.Background(), "participant-documents", "p-1/ktp.jpg"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !strings.HasSuffix(deleted, "/participant-documents/p-1/ktp.jpg") {
		t.Errorf("objek yang dihapus = %q", deleted)
	}

	status = http.StatusForbidden
	if err := svc.Delete(context.Background(), "participant-documents", "p-1/ktp.jpg"); err == nil {
		t.Error("penolakan dari penyimpanan tidak dilaporkan")
	}
}

// fileHeaderFor builds the multipart header an upload handler would hand over.
func fileHeaderFor(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	var body strings.Builder
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("buat form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("tulis isi: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("tutup writer: %v", err)
	}

	reader := multipart.NewReader(strings.NewReader(body.String()), w.Boundary())
	form, err := reader.ReadForm(10 << 20)
	if err != nil {
		t.Fatalf("baca form: %v", err)
	}
	t.Cleanup(func() { _ = form.RemoveAll() })
	return form.File["file"][0]
}
