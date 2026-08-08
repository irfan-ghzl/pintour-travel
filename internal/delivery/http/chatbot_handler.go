package httpdelivery

import (
	"context"
	"log"
	"net/http"
	"strings"

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

	var phone, message string
	if strings.Contains(c.Request().Header.Get("Content-Type"), "application/json") {
		var b struct {
			Phone   string `json:"phone"`
			Sender  string `json:"sender"`
			Message string `json:"message"`
		}
		_ = bindJSON(c, &b)
		phone, message = firstNonEmpty(b.Phone, b.Sender), b.Message
	} else {
		phone = firstNonEmpty(c.FormValue("sender"), c.FormValue("phone"))
		message = c.FormValue("message")
	}

	if phone == "" || message == "" {
		return c.JSON(http.StatusOK, ok(map[string]string{"status": "ignored"}))
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
