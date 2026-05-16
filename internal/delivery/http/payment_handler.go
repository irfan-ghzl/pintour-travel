package httpdelivery

import (
	"net/http"

	paymentsvc "github.com/irfan-ghzl/pintour-travel/internal/application/payment"
	"github.com/irfan-ghzl/pintour-travel/internal/auth"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/payment"
	"github.com/labstack/echo/v4"
)

// PaymentHandler handles HTTP requests for payment management.
type PaymentHandler struct {
	svc *paymentsvc.PaymentService
}

// NewPaymentHandler creates a new PaymentHandler.
func NewPaymentHandler(svc *paymentsvc.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

// ListPayments godoc
//
//	@Summary     List payments for a booking (admin)
//	@Tags        payments
//	@Security    BearerAuth
//	@Param       id path string true "Booking ID"
//	@Success     200 {array} payment.Payment
//	@Router      /api/v1/admin/bookings/{id}/payments [get]
func (h *PaymentHandler) ListPayments(c echo.Context) error {
	bookingID := c.Param("id")
	payments, err := h.svc.ListByBooking(c.Request().Context(), bookingID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch payments")
	}
	if payments == nil {
		payments = []payment.Payment{}
	}
	return c.JSON(http.StatusOK, payments)
}

// CreatePayment godoc
//
//	@Summary     Record a new payment for a booking (admin)
//	@Tags        payments
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id path string true "Booking ID"
//	@Success     201 {object} map[string]interface{}
//	@Router      /api/v1/admin/bookings/{id}/payments [post]
func (h *PaymentHandler) CreatePayment(c echo.Context) error {
	bookingID := c.Param("id")
	var body struct {
		PaymentType string  `json:"payment_type"`
		Amount      float64 `json:"amount"`
		PaidAt      string  `json:"paid_at"`
		ProofURL    string  `json:"proof_url"`
		Notes       string  `json:"notes"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if !paymentsvc.ValidPaymentTypes[body.PaymentType] {
		return echo.NewHTTPError(http.StatusBadRequest, "payment_type must be one of: dp, pelunasan, full")
	}
	if body.Amount <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "amount must be greater than 0")
	}
	if body.PaidAt == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "paid_at is required")
	}

	params := payment.CreateParams{
		BookingID:   bookingID,
		PaymentType: body.PaymentType,
		Amount:      body.Amount,
		PaidAt:      body.PaidAt,
	}
	if body.ProofURL != "" {
		params.ProofURL = &body.ProofURL
	}
	if body.Notes != "" {
		params.Notes = &body.Notes
	}

	id, err := h.svc.CreatePayment(c.Request().Context(), params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create payment")
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{"id": id})
}

// VerifyPayment godoc
//
//	@Summary     Mark a payment as verified (admin)
//	@Tags        payments
//	@Security    BearerAuth
//	@Param       pid path string true "Payment ID"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/payments/{pid}/verify [patch]
func (h *PaymentHandler) VerifyPayment(c echo.Context) error {
	pid := c.Param("pid")
	claims := c.Get("claims").(*auth.Claims)
	if err := h.svc.VerifyPayment(c.Request().Context(), pid, claims.UserID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify payment")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"id": pid, "verified": true})
}

// DeletePayment godoc
//
//	@Summary     Delete a payment (admin)
//	@Tags        payments
//	@Security    BearerAuth
//	@Param       pid path string true "Payment ID"
//	@Success     204
//	@Router      /api/v1/admin/payments/{pid} [delete]
func (h *PaymentHandler) DeletePayment(c echo.Context) error {
	if err := h.svc.DeletePayment(c.Request().Context(), c.Param("pid")); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete payment")
	}
	return c.NoContent(http.StatusNoContent)
}
