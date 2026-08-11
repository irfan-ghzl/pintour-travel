package httpdelivery

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	leadsvc "github.com/irfan-ghzl/pintour-travel/internal/application/lead"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/chatbot"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	"github.com/irfan-ghzl/pintour-travel/internal/safe"
	"github.com/irfan-ghzl/pintour-travel/internal/service"
)

// ChatbotHandler handles the Fonnte inbound webhook (public) and the admin
// chatbot-log endpoints (v2.0 F2).
type ChatbotHandler struct {
	svc          *service.ChatbotService
	logs         chatbot.Repository
	leads        *leadsvc.Service
	webhookToken string
}

func NewChatbotHandler(svc *service.ChatbotService, logs chatbot.Repository, leads *leadsvc.Service, webhookToken string) *ChatbotHandler {
	return &ChatbotHandler{svc: svc, logs: logs, leads: leads, webhookToken: webhookToken}
}

// HandleFonnteWebhook receives an inbound WA message from Fonnte (public).
//
//	@Summary  Webhook pesan masuk Fonnte → chatbot AI
//	@Tags     webhooks
//	@Success  200 {object} map[string]interface{}
//	@Router   /webhooks/fonnte [post]
func (h *ChatbotHandler) HandleFonnteWebhook(c echo.Context) error {
	// Validate the shared token (skip only when no token is configured — dev).
	if h.webhookToken != "" {
		tok := c.QueryParam("token")
		if tok == "" {
			tok = c.Request().Header.Get("X-Token")
		}
		if tok != h.webhookToken {
			return c.JSON(http.StatusUnauthorized, errResponse("UNAUTHORIZED", "token webhook tidak valid"))
		}
	}

	var phone, message, device string
	if strings.Contains(c.Request().Header.Get("Content-Type"), "application/json") {
		var b struct {
			Phone   string `json:"phone"`
			Sender  string `json:"sender"`
			Message string `json:"message"`
			Device  string `json:"device"`
		}
		_ = bindJSON(c, &b)
		phone, message, device = firstNonEmpty(b.Phone, b.Sender), b.Message, b.Device
	} else {
		phone = firstNonEmpty(c.FormValue("sender"), c.FormValue("phone"))
		message = c.FormValue("message")
		device = c.FormValue("device")
	}

	if phone == "" || message == "" {
		return c.JSON(http.StatusOK, ok(map[string]string{"status": "ignored"}))
	}

	// Pesan yang dikirim perangkat kita sendiri diabaikan.
	//
	// Fonnte meneruskan SELURUH lalu lintas perangkat ke webhook, termasuk pesan
	// keluar. Tanpa penjagaan ini, balasan bot kembali masuk sebagai pesan baru
	// dan dibalas lagi — dan seterusnya. Terjadi sungguhan: 35 balasan dalam dua
	// menit sampai tunnel dimatikan dengan tangan.
	if device != "" && normalizePhone(device) == normalizePhone(phone) {
		return c.JSON(http.StatusOK, ok(map[string]string{"status": "self_echo_ignored"}))
	}

	// Jaring pengaman yang tidak bergantung pada bentuk payload.
	//
	// Penjagaan di atas mengandalkan Fonnte mengirim field `device`; bila suatu
	// saat tidak, atau gemanya datang lewat jalur lain, loop akan berjalan lagi
	// tanpa ada yang menghentikannya. Batas ini tidak mungkin tersentuh percakapan
	// manusia, tapi memotong loop dalam hitungan detik.
	if chatbotFlood.tripped(normalizePhone(phone)) {
		log.Printf("chatbot: %s melewati batas laju — kemungkinan loop, balasan dihentikan", phone)
		return c.JSON(http.StatusOK, ok(map[string]string{"status": "rate_limited"}))
	}
	if h.svc == nil || !h.svc.Active() {
		return c.JSON(http.StatusOK, ok(map[string]string{"status": "chatbot_inactive"}))
	}

	// Process asynchronously so we ack Fonnte immediately.
	p, m := phone, message
	safe.Go("balas pesan chatbot masuk", func() {
		if err := h.svc.HandleIncomingMessage(context.Background(), normalizePhone(p), m); err != nil {
			log.Printf("chatbot: gagal proses pesan masuk dari %s: %v", p, err)
		}
	})

	return c.JSON(http.StatusOK, ok(map[string]string{"status": "received"}))
}

// ListChatbotConversations godoc
//
//	@Summary  Daftar percakapan chatbot per nomor (admin)
//	@Tags     chatbot
//	@Security BearerAuth
//	@Router   /admin/chatbot-logs [get]
func (h *ChatbotHandler) ListConversations(c echo.Context) error {
	f := chatbot.Filter{
		Phone:    c.QueryParam("phone"),
		DateFrom: c.QueryParam("from"),
		DateTo:   c.QueryParam("to"),
		Page:     queryInt(c, "page", 1),
		Limit:    queryPageSize(c, "limit", 20),
	}
	list, total, err := h.logs.ListConversations(c.Request().Context(), f)
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, pageResponse(list, total, f.Page, f.Limit))
}

// GetConversation godoc
//
//	@Summary  Detail percakapan chatbot satu nomor (admin)
//	@Tags     chatbot
//	@Security BearerAuth
//	@Router   /admin/chatbot-logs/{phone} [get]
func (h *ChatbotHandler) GetConversation(c echo.Context) error {
	logs, err := h.logs.ListByPhone(c.Request().Context(), c.Param("phone"))
	if err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusOK, ok(logs))
}

// CreateLeadFromChat godoc
//
//	@Summary  Buat leads manual dari percakapan chatbot (admin)
//	@Tags     chatbot
//	@Security BearerAuth
//	@Router   /admin/chatbot-logs/{phone}/create-lead [post]
func (h *ChatbotHandler) CreateLeadFromChat(c echo.Context) error {
	var body struct {
		Name      string `json:"name" validate:"required"`
		PackageID string `json:"package_id" validate:"required"`
		Pax       int    `json:"pax" validate:"omitempty,gte=1"`
	}
	if err := bindJSON(c, &body); err != nil {
		return invalidPayload(c, err, "name dan package_id harus diisi")
	}
	if body.Pax < 1 {
		body.Pax = 1
	}
	l := &domainLead.Lead{
		Name:      body.Name,
		Phone:     c.Param("phone"),
		PackageID: body.PackageID,
		Pax:       body.Pax,
		Source:    "chatbot",
		Message:   "Dibuat manual dari percakapan chatbot",
	}
	if err := h.leads.CreateLead(c.Request().Context(), l); err != nil {
		return serverErr(c, err)
	}
	return c.JSON(http.StatusCreated, ok(l))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ─── Penjaga loop chatbot ─────────────────────────────────────────────────────

// chatbotFlood memutus percakapan yang membalas terlalu cepat untuk bisa
// dilakukan manusia.
//
// Delapan pesan per menit dari satu nomor sudah jauh di atas percakapan wajar,
// sementara loop gema menghasilkan sekitar tiga puluh. Batas ini bukan pengganti
// penjagaan self-echo di atas melainkan lapisan kedua: yang pertama bergantung
// pada Fonnte mengirim field `device`, yang ini tidak bergantung pada apa pun.
var chatbotFlood = &floodGuard{limit: 8, window: time.Minute}

type floodGuard struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	seen   map[string]*floodCounter
}

type floodCounter struct {
	count  int
	expiry time.Time
}

// tripped mencatat satu pesan dari key dan melaporkan apakah batasnya terlampaui.
func (g *floodGuard) tripped(key string) bool {
	now := time.Now()

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.seen == nil {
		g.seen = map[string]*floodCounter{}
	}

	// Nomor yang pernah menghubungi tidak boleh menumpuk selamanya di memori.
	// Disapu saat petanya membesar, bukan lewat goroutine tersendiri — pekerjaan
	// sekecil ini tidak sebanding dengan biaya menjaga satu goroutine hidup.
	if len(g.seen) > 1000 {
		for k, c := range g.seen {
			if now.After(c.expiry) {
				delete(g.seen, k)
			}
		}
	}

	c := g.seen[key]
	if c == nil || now.After(c.expiry) {
		g.seen[key] = &floodCounter{count: 1, expiry: now.Add(g.window)}
		return false
	}
	c.count++
	return c.count > g.limit
}
