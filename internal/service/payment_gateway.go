package service

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/format"
)

// MidtransService is a thin Midtrans Snap client (v2.0 F1). Following the same
// pattern as FonnteService/EmailService, it talks to the Snap REST API directly
// over net/http instead of pulling in an SDK.
type MidtransService struct {
	serverKey string
	clientKey string
	baseURL   string
	client    *http.Client
}

// MidtransOption customizes the client at construction.
type MidtransOption func(*MidtransService)

// WithMidtransBaseURL overrides the Snap API base URL. Sandbox and production
// keep the defaults below; tests point this at a local httptest server so the
// transaction-creation path can be exercised without reaching Midtrans.
func WithMidtransBaseURL(baseURL string) MidtransOption {
	return func(s *MidtransService) {
		if baseURL != "" {
			s.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

func NewMidtransService(serverKey, clientKey, env string, opts ...MidtransOption) *MidtransService {
	base := "https://app.sandbox.midtrans.com"
	if env == "production" {
		base = "https://app.midtrans.com"
	}
	s := &MidtransService{
		serverKey: serverKey,
		clientKey: clientKey,
		baseURL:   base,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Enabled reports whether Midtrans is configured.
func (s *MidtransService) Enabled() bool { return s != nil && s.serverKey != "" }

// ClientKey returns the public client key (sent to the browser for Snap.js).
func (s *MidtransService) ClientKey() string {
	if s == nil {
		return ""
	}
	return s.clientKey
}

// SnapCustomer holds the buyer details shown in the Snap UI.
type SnapCustomer struct {
	Name  string
	Email string
	Phone string
}

// SnapRequest is the input for CreateSnap. GrossAmount is in whole rupiah.
type SnapRequest struct {
	OrderID     string
	GrossAmount int64
	ItemName    string
	Customer    SnapCustomer
	ExpiryHours int
}

// CreateSnap creates a Snap transaction and returns its token.
func (s *MidtransService) CreateSnap(ctx context.Context, req SnapRequest) (string, error) {
	if !s.Enabled() {
		return "", fmt.Errorf("midtrans belum dikonfigurasi (MIDTRANS_SERVER_KEY kosong)")
	}
	if req.ExpiryHours <= 0 {
		req.ExpiryHours = 24
	}
	payload := map[string]interface{}{
		"transaction_details": map[string]interface{}{
			"order_id":     req.OrderID,
			"gross_amount": req.GrossAmount,
		},
		"customer_details": map[string]interface{}{
			"first_name": req.Customer.Name,
			"email":      req.Customer.Email,
			"phone":      req.Customer.Phone,
		},
		"item_details": []map[string]interface{}{{
			"id":       req.OrderID,
			"price":    req.GrossAmount,
			"quantity": 1,
			"name":     format.TruncateRunes(req.ItemName, 50), // Midtrans name max 50 chars
		}},
		"expiry": map[string]interface{}{
			"unit":     "hours",
			"duration": req.ExpiryHours,
		},
	}
	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.baseURL+"/snap/v1/transactions", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	// Basic auth: username = server key, password empty.
	httpReq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(s.serverKey+":")))

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out struct {
		Token         string   `json:"token"`
		ErrorMessages []string `json:"error_messages"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode >= 400 || out.Token == "" {
		if len(out.ErrorMessages) > 0 {
			return "", fmt.Errorf("midtrans: %s", strings.Join(out.ErrorMessages, "; "))
		}
		return "", fmt.Errorf("midtrans snap error: status %d", resp.StatusCode)
	}
	return out.Token, nil
}

// VerifySignature validates the Midtrans webhook signature:
// sha512(order_id + status_code + gross_amount + server_key), compared in
// constant time. Per Midtrans docs the signature is lowercase hex.
func (s *MidtransService) VerifySignature(orderID, statusCode, grossAmount, signatureKey string) bool {
	if s == nil || s.serverKey == "" {
		return false
	}
	sum := sha512.Sum512([]byte(orderID + statusCode + grossAmount + s.serverKey))
	expected := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(strings.ToLower(signatureKey))) == 1
}
