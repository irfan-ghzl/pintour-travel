package service

// Ticket 13 — the WhatsApp assistant (v2.0 F2), against a stand-in model.
//
// What matters to a customer messaging Pintour is that they always get an
// answer: the assistant either relays what the model said, or — when the model
// is rate-limited, slow, or broken — sends a holding message so a consultant
// can pick the conversation up. Both halves are asserted here, along with the
// prompt the model is actually given.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/chatbot"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
)

// ─── Fakes ────────────────────────────────────────────────────────────────────

type fakeChatLogRepo struct {
	mu   sync.Mutex
	logs []chatbot.Log
}

func (r *fakeChatLogRepo) Create(_ context.Context, l *chatbot.Log) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, *l)
	return nil
}

func (r *fakeChatLogRepo) RecentByPhone(_ context.Context, phone string, limit int) ([]chatbot.Log, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []chatbot.Log
	for _, l := range r.logs {
		if l.Phone == phone {
			out = append(out, l)
		}
	}
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func (r *fakeChatLogRepo) ListConversations(_ context.Context, _ chatbot.Filter) ([]chatbot.Conversation, int, error) {
	return nil, 0, nil
}

func (r *fakeChatLogRepo) ListByPhone(_ context.Context, phone string) ([]chatbot.Log, error) {
	return r.RecentByPhone(context.Background(), phone, 1000)
}

// replies returns just the assistant's turns.
func (r *fakeChatLogRepo) replies() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, l := range r.logs {
		if l.Role == "assistant" {
			out = append(out, l.Message)
		}
	}
	return out
}

type fakeCatalogue struct{ packages []domainPkg.Package }

func (r *fakeCatalogue) Create(context.Context, *domainPkg.Package) error { return nil }
func (r *fakeCatalogue) Update(context.Context, *domainPkg.Package) error { return nil }
func (r *fakeCatalogue) Delete(context.Context, string) error             { return nil }
func (r *fakeCatalogue) GetByID(context.Context, string) (*domainPkg.Package, error) {
	return nil, nil
}
func (r *fakeCatalogue) GetBySlug(context.Context, string) (*domainPkg.Package, error) {
	return nil, nil
}
func (r *fakeCatalogue) List(_ context.Context, _ domainPkg.Filter) ([]domainPkg.Package, int, error) {
	return r.packages, len(r.packages), nil
}

// ─── Harness ──────────────────────────────────────────────────────────────────

// geminiStub answers generateContent with whatever the test scripted, and keeps
// the prompt it was given.
type geminiStub struct {
	mu       sync.Mutex
	requests []geminiRequest
	srv      *httptest.Server
}

func (g *geminiStub) prompt(t *testing.T) string {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.requests) == 0 {
		t.Fatal("model tidak pernah dipanggil")
	}
	req := g.requests[0]
	if req.SystemInstruction == nil || len(req.SystemInstruction.Parts) == 0 {
		t.Fatal("permintaan ke model tanpa instruksi sistem")
	}
	return req.SystemInstruction.Parts[0].Text
}

func (g *geminiStub) calls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.requests)
}

// newChatbot wires an assistant onto a scripted model. handler decides what the
// model says on each call.
func newChatbot(t *testing.T, handler http.HandlerFunc) (*ChatbotService, *fakeChatLogRepo, *waRecorder, *geminiStub) {
	t.Helper()
	stub := &geminiStub{}
	stub.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var parsed geminiRequest
		_ = json.Unmarshal(body, &parsed)
		stub.mu.Lock()
		stub.requests = append(stub.requests, parsed)
		stub.mu.Unlock()
		handler(w, r)
	}))
	t.Cleanup(stub.srv.Close)

	fonnte, wa, _ := newWARecorder(t)
	logs := &fakeChatLogRepo{}
	catalogue := &fakeCatalogue{packages: []domainPkg.Package{
		{ID: "package-1", Name: "Umroh Reguler 9 Hari", Destination: "Arab Saudi",
			BasePrice: 25000000, DurationDays: 9},
		{ID: "package-2", Name: "Halal Tour Jepang", Destination: "Jepang",
			BasePrice: 32000000, DurationDays: 7},
	}}

	svc := NewChatbotService(logs, nil, catalogue, fonnte, "test-key", "", 10, true,
		WithGeminiBaseURL(stub.srv.URL))
	return svc, logs, wa, stub
}

// replyWith writes a well-formed Gemini answer.
func replyWith(text string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"parts": []map[string]any{{"text": text}}}},
			},
		})
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// A customer asks a question and gets the assistant's answer over WhatsApp,
// with both turns recorded in the log an admin can read later.
func TestChatbot_AnswersAndRecordsBothTurns(t *testing.T) {
	svc, logs, wa, _ := newChatbot(t, replyWith("Ada, batch 10 Desember masih tersedia."))

	if err := svc.HandleIncomingMessage(context.Background(),
		"628444444444", "Ada paket umroh Desember?"); err != nil {
		t.Fatalf("HandleIncomingMessage: %v", err)
	}

	if got := logs.replies(); len(got) != 1 || got[0] != "Ada, batch 10 Desember masih tersedia." {
		t.Errorf("balasan tercatat = %v", got)
	}
	payload := wa.last(t)
	if got, _ := payload["target"].(string); got != "628444444444" {
		t.Errorf("balasan dikirim ke %q, want 628444444444", got)
	}
	if msg, _ := payload["message"].(string); !strings.Contains(msg, "10 Desember") {
		t.Errorf("pesan terkirim = %q", msg)
	}
}

// The live catalogue goes into the prompt, so the assistant quotes packages
// that exist at prices that are current rather than whatever it remembers.
func TestChatbot_PromptCarriesTheLiveCatalogue(t *testing.T) {
	svc, _, _, stub := newChatbot(t, replyWith("Baik kak."))

	_ = svc.HandleIncomingMessage(context.Background(), "628444444444", "Ada paket apa saja?")

	prompt := stub.prompt(t)
	for _, want := range []string{"Umroh Reguler 9 Hari", "Halal Tour Jepang", "Arab Saudi"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("instruksi sistem tidak memuat %q", want)
		}
	}
}

// A model that fails is retried once, and when it fails again the customer
// still gets a holding message instead of silence.
func TestChatbot_FallsBackWhenTheModelFails(t *testing.T) {
	svc, logs, wa, stub := newChatbot(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"internal"}}`))
	})

	if err := svc.HandleIncomingMessage(context.Background(),
		"628444444444", "Ada paket umroh Desember?"); err != nil {
		t.Fatalf("HandleIncomingMessage: %v", err)
	}

	if stub.calls() != 2 {
		t.Errorf("panggilan ke model = %d, want 2 (satu percobaan ulang)", stub.calls())
	}
	if got := logs.replies(); len(got) != 1 || got[0] != chatbotFallbackReply {
		t.Errorf("balasan = %v, want pesan cadangan", got)
	}
	if msg, _ := wa.last(t)["message"].(string); msg != chatbotFallbackReply {
		t.Error("pesan cadangan tidak sampai ke pelanggan")
	}
}

// A model answering with nothing at all counts as a failure, not as an answer:
// an empty WhatsApp message reads as the system being broken.
func TestChatbot_EmptyAnswerFallsBackToo(t *testing.T) {
	svc, logs, _, _ := newChatbot(t, replyWith(""))

	_ = svc.HandleIncomingMessage(context.Background(), "628444444444", "Halo")

	if got := logs.replies(); len(got) != 1 || got[0] != chatbotFallbackReply {
		t.Errorf("balasan = %v, want pesan cadangan", got)
	}
}

// The first attempt succeeding means no second one — the retry is for failure,
// not a habit.
func TestChatbot_SuccessDoesNotRetry(t *testing.T) {
	svc, _, _, stub := newChatbot(t, replyWith("Halo kak!"))

	_ = svc.HandleIncomingMessage(context.Background(), "628444444444", "Halo")

	if stub.calls() != 1 {
		t.Errorf("panggilan ke model = %d, want 1", stub.calls())
	}
}

// An assistant that is switched off, or has no key, does nothing at all — a
// deployment without Gemini must not answer customers with an error.
func TestChatbot_InactiveStaysSilent(t *testing.T) {
	cases := map[string]struct {
		apiKey string
		active bool
	}{
		"dinonaktifkan": {"test-key", false},
		"tanpa API key": {"", true},
		"keduanya mati": {"", false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			logs := &fakeChatLogRepo{}
			svc := NewChatbotService(logs, nil, &fakeCatalogue{}, nil,
				tc.apiKey, "", 10, tc.active)
			if svc.Active() {
				t.Fatal("Active() = true, want false")
			}
			if err := svc.HandleIncomingMessage(context.Background(), "628444444444", "Halo"); err != nil {
				t.Errorf("HandleIncomingMessage = %v, want nil", err)
			}
			if len(logs.logs) != 0 {
				t.Errorf("tercatat %d baris meski chatbot mati", len(logs.logs))
			}
		})
	}
}

// The conversation history is replayed to the model so a follow-up question
// makes sense — and Gemini's own vocabulary is used for the roles.
func TestChatbot_ReplaysHistoryInTheModelsVocabulary(t *testing.T) {
	svc, logs, _, stub := newChatbot(t, replyWith("Batch 10 Desember, kak."))
	logs.logs = []chatbot.Log{
		{Phone: "628444444444", Role: "user", Message: "Ada paket umroh?"},
		{Phone: "628444444444", Role: "assistant", Message: "Ada beberapa, kak."},
	}

	_ = svc.HandleIncomingMessage(context.Background(), "628444444444", "Yang Desember?")

	stub.mu.Lock()
	defer stub.mu.Unlock()
	contents := stub.requests[0].Contents
	if len(contents) < 3 {
		t.Fatalf("riwayat terkirim = %d giliran, want minimal 3", len(contents))
	}
	if contents[0].Role != "user" {
		t.Errorf("giliran pertama = %q, want user", contents[0].Role)
	}
	if contents[1].Role != "model" {
		t.Errorf("giliran asisten dikirim sebagai %q, want model", contents[1].Role)
	}
}

// Kehabisan kuota tidak dicoba ulang. Batas paket gratis dihitung per menit dan
// Gemini menyebutkan sendiri bahwa ia baru mau dilayani puluhan detik kemudian;
// mengulang 1,5 detik lagi pasti gagal, sementara kegagalannya tetap dihitung
// sebagai satu permintaan. Percobaan ulang di sini menggandakan laju pemakaian
// persis ketika lajunya yang sedang jadi masalah.
func TestChatbot_QuotaExceededIsNotRetried(t *testing.T) {
	svc, logs, wa, stub := newChatbot(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"You exceeded your current quota"}}`))
	})

	if err := svc.HandleIncomingMessage(context.Background(),
		"628444444444", "Ada paket umroh Desember?"); err != nil {
		t.Fatalf("HandleIncomingMessage: %v", err)
	}

	if stub.calls() != 1 {
		t.Errorf("panggilan ke model = %d, want 1 (kuota habis tidak diulang)", stub.calls())
	}
	// Pelanggan tetap dijawab; yang hilang hanya jawabannya, bukan balasannya.
	if got := logs.replies(); len(got) != 1 || got[0] != chatbotFallbackReply {
		t.Errorf("balasan = %v, want pesan cadangan", got)
	}
	if msg, _ := wa.last(t)["message"].(string); msg != chatbotFallbackReply {
		t.Error("pesan cadangan tidak sampai ke pelanggan")
	}
}
