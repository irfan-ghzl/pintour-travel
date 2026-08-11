package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/chatbot"
	domainPkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
)

// defaultGeminiBaseURL is where the assistant asks for a reply in production.
const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com"

// chatbotFallbackReply is sent when Gemini fails (rate limit / timeout / error)
// so the customer never hits a silent dead-end and a consultant can follow up.
const chatbotFallbackReply = "Halo! 👋 Terima kasih sudah menghubungi Pintour. " +
	"Mohon tunggu sebentar ya, tim konsultan kami akan segera membalas pesan Anda. 🙏"

// ChatbotService handles inbound WA messages with a Gemini-powered assistant
// (v2.0 F2). It keeps conversation history, injects live package data into the
// system prompt, recognizes returning customers (F5), and replies via Fonnte.
type ChatbotService struct {
	logs       chatbot.Repository
	db         *sql.DB
	packages   domainPkg.Repository
	fonnte     *FonnteService
	apiKey     string
	model      string
	maxHistory int
	active     bool
	baseURL    string
	client     *http.Client
}

// ChatbotOption customizes a ChatbotService at construction.
type ChatbotOption func(*ChatbotService)

// WithGeminiBaseURL points the assistant at a different host — a test server,
// which is what lets the retry, the fallback reply and the prompt assembly be
// exercised without a live model. Same shape as WithFonnteBaseURL.
func WithGeminiBaseURL(baseURL string) ChatbotOption {
	return func(s *ChatbotService) {
		if baseURL != "" {
			s.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

func NewChatbotService(
	logs chatbot.Repository,
	db *sql.DB,
	packages domainPkg.Repository,
	fonnte *FonnteService,
	apiKey, model string,
	maxHistory int,
	active bool,
	opts ...ChatbotOption,
) *ChatbotService {
	if model == "" {
		model = "gemini-1.5-flash"
	}
	if maxHistory <= 0 {
		maxHistory = 10
	}
	s := &ChatbotService{
		logs:       logs,
		db:         db,
		packages:   packages,
		fonnte:     fonnte,
		apiKey:     apiKey,
		model:      model,
		maxHistory: maxHistory,
		active:     active,
		baseURL:    defaultGeminiBaseURL,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Active reports whether the chatbot is enabled and configured.
func (s *ChatbotService) Active() bool {
	return s != nil && s.active && s.apiKey != ""
}

// HandleIncomingMessage processes one inbound WA message end to end.
func (s *ChatbotService) HandleIncomingMessage(ctx context.Context, phone, message string) error {
	if !s.Active() {
		return nil
	}
	// 1. Log the user message.
	_ = s.logs.Create(ctx, &chatbot.Log{Phone: phone, Role: "user", Message: message})

	// 2. If a consultant is already handling this number, stay out of the way.
	//
	// Guarded on the connection the same way returningContext below already is:
	// the two reads are both best-effort colour on the conversation, and one of
	// them dereferencing a nil connection while the other checks for it is an
	// inconsistency, not a design.
	if s.db != nil {
		var handled bool
		_ = s.db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM leads WHERE phone=$1 AND status IN ('konsultasi','deal') AND deleted_at IS NULL)`,
			phone).Scan(&handled)
		if handled {
			return nil
		}
	}

	// 3. Conversation history (already ends with the message we just stored).
	history, err := s.logs.RecentByPhone(ctx, phone, s.maxHistory)
	if err != nil {
		return err
	}

	// 4. Active packages for context.
	active := true
	pkgs, _, _ := s.packages.List(ctx, domainPkg.Filter{IsActive: &active, Page: 1, PerPage: 10})

	// 4b. Returning-customer context (F5) — empty for new numbers.
	returningInfo := s.returningContext(ctx, phone)

	// 5. Generate reply. On failure, fall back to a graceful holding message so
	// the customer always gets a response and a consultant can pick it up.
	reply, err := s.generateReply(ctx, history, pkgs, returningInfo)
	if err != nil {
		log.Printf("chatbot: Gemini gagal untuk %s: %v — kirim balasan fallback", phone, err)
		reply = chatbotFallbackReply
	}

	// 6. Log + send the assistant reply (hasil Gemini atau fallback).
	_ = s.logs.Create(ctx, &chatbot.Log{Phone: phone, Role: "assistant", Message: reply})
	refType := "chatbot"
	return s.fonnte.Send(ctx, phone, "", "CHATBOT_REPLY", reply, nil, &refType)
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  map[string]int  `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// returningContext returns a short note about the sender when their number
// already belongs to a portal account (F5): name + their latest tour. Empty for
// new numbers or on any error (best-effort, must never block the reply).
func (s *ChatbotService) returningContext(ctx context.Context, phone string) string {
	if s.db == nil {
		return ""
	}
	var name, lastPackage string
	var departure sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT pu.name, COALESCE(pkg.name,''), pb.departure_date
		FROM portal_users pu
		LEFT JOIN LATERAL (
			SELECT pt.batch_id FROM participants pt
			WHERE (pt.portal_user_id = pu.id OR pt.phone = pu.phone)
			  AND pt.deleted_at IS NULL
			ORDER BY pt.created_at DESC LIMIT 1
		) lastpt ON true
		LEFT JOIN package_batches pb ON pb.id = lastpt.batch_id
		LEFT JOIN packages pkg ON pkg.id = pb.package_id
		WHERE pu.phone = $1`, phone,
	).Scan(&name, &lastPackage, &departure)
	if err != nil || name == "" {
		return ""
	}
	info := fmt.Sprintf("Pengirim adalah pelanggan lama Pintour bernama %s. Sapa dengan nama tersebut.", name)
	if lastPackage != "" {
		info += fmt.Sprintf(" Tour terakhirnya: %s", lastPackage)
		if departure.Valid {
			info += fmt.Sprintf(" (berangkat %s)", departure.Time.Format("02 January 2006"))
		}
		info += "."
	}
	return info
}

func (s *ChatbotService) generateReply(ctx context.Context, history []chatbot.Log, pkgs []domainPkg.Package, returningInfo string) (string, error) {
	var contents []geminiContent
	for _, h := range history {
		if len(contents) == 0 && h.Role != "user" {
			continue
		}
		role := h.Role
		if role == "assistant" {
			role = "model" // Gemini pakai "model" bukan "assistant"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: h.Message}},
		})
	}
	if len(contents) == 0 {
		return "", fmt.Errorf("tidak ada pesan user")
	}

	reqBody := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: buildSystemPrompt(pkgs, returningInfo)}},
		},
		Contents:         contents,
		GenerationConfig: map[string]int{"maxOutputTokens": 300},
	}
	body, _ := json.Marshal(reqBody)

	model := s.model
	if model == "" {
		model = "gemini-1.5-flash"
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", s.baseURL, model, s.apiKey)

	// Retry once on transient failure (timeout / empty candidate), which is the
	// intermittent behaviour observed on the free tier.
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		reply, err := s.callGemini(ctx, url, body)
		if err == nil && reply != "" {
			return reply, nil
		}
		lastErr = err
		if err == nil {
			lastErr = fmt.Errorf("gemini: balasan kosong")
		}

		// Kehabisan kuota dikecualikan dari percobaan ulang.
		//
		// Batas paket gratis dihitung per menit, dan Gemini menyebutkan sendiri
		// berapa detik lagi ia mau dilayani — biasanya sekitar empat puluh.
		// Mengulang 1,5 detik kemudian pasti gagal, dan kegagalan itu tetap
		// dihitung sebagai satu permintaan. Jadi percobaan ulang di sini bukan
		// sekadar sia-sia: ia menggandakan laju pemakaian persis ketika lajunya
		// yang sedang jadi masalah, dan memperpanjang durasi kehabisan kuota.
		if isQuotaExceeded(lastErr) {
			return "", lastErr
		}

		if attempt < 2 {
			time.Sleep(1500 * time.Millisecond)
		}
	}
	return "", lastErr
}

// isQuotaExceeded melaporkan apakah kegagalan berasal dari batas laju, bukan
// dari gangguan sesaat yang layak dicoba ulang.
func isQuotaExceeded(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 429") ||
		strings.Contains(msg, "resource_exhausted") ||
		strings.Contains(msg, "exceeded your current quota")
}

// callGemini performs a single generateContent request and extracts the text.
func (s *ChatbotService) callGemini(ctx context.Context, url string, body []byte) (string, error) {
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

	var out geminiResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Error != nil {
		// Kode status ikut dibawa. Pesan dari Gemini berguna untuk dibaca manusia,
		// tapi yang menentukan boleh-tidaknya dicoba ulang adalah statusnya, dan
		// menebaknya dari kalimat berarti bergantung pada susunan kata yang bisa
		// berubah kapan saja.
		return "", fmt.Errorf("gemini: status %d: %s", resp.StatusCode, out.Error.Message)
	}
	if resp.StatusCode >= 400 || len(out.Candidates) == 0 {
		return "", fmt.Errorf("gemini error: status %d", resp.StatusCode)
	}
	var sb strings.Builder
	for _, p := range out.Candidates[0].Content.Parts {
		sb.WriteString(p.Text)
	}
	return strings.TrimSpace(sb.String()), nil
}

func buildSystemPrompt(pkgs []domainPkg.Package, returningInfo string) string {
	var list strings.Builder
	for _, p := range pkgs {
		list.WriteString(fmt.Sprintf("- %s | %s | mulai dari Rp %.0f | %d hari\n",
			p.Name, p.Destination, p.BasePrice, p.DurationDays))
	}
	if list.Len() == 0 {
		list.WriteString("(belum ada paket aktif)\n")
	}
	customerBlock := ""
	if returningInfo != "" {
		customerBlock = "\nTENTANG PENGIRIM:\n" + returningInfo + "\n"
	}
	return `Kamu adalah asisten virtual Pintour, agen tour & travel terpercaya.
Tugasmu menjawab pertanyaan awal calon pelanggan dengan ramah dan informatif.

ATURAN:
- Jawab singkat, maksimal 3-4 kalimat.
- Gunakan Bahasa Indonesia yang sopan dan hangat.
- Gunakan emoji secukupnya agar terasa personal.
- JANGAN berikan harga pasti, selalu sebut "mulai dari Rp X".
- JANGAN buat janji yang tidak bisa dipenuhi.
` + customerBlock + `
PAKET TERSEDIA:
` + list.String() + `
SKENARIO:
- Tanya harga/paket -> jelaskan paket yang relevan.
- Minta daftar/booking -> minta nama lengkap dan nomor HP aktif.
- Sudah kasih nama + nomor -> konfirmasi dan beritahu konsultan akan menghubungi.
- Pertanyaan di luar wisata -> arahkan ke konsultan.
- Tidak relevan -> tetap sopan, tawarkan bantuan lain.`
}
