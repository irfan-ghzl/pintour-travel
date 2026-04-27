package httpdelivery

import (
	"database/sql"
	"net/http"

	bookingsvc "github.com/irfan-ghzl/pintour-travel/internal/application/booking"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/booking"
	"github.com/labstack/echo/v4"
)

// BookingHandler handles HTTP requests for bookings and manifest.
type BookingHandler struct {
	svc *bookingsvc.BookingService
}

// NewBookingHandler creates a new BookingHandler.
func NewBookingHandler(svc *bookingsvc.BookingService) *BookingHandler {
	return &BookingHandler{svc: svc}
}

// ListBookings godoc
//
//	@Summary     List bookings (admin)
//	@Tags        bookings
//	@Produce     json
//	@Security    BearerAuth
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/bookings [get]
func (h *BookingHandler) ListBookings(c echo.Context) error {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 15)
	if perPage > 100 {
		perPage = 100
	}

	f := booking.Filter{
		PaymentStatus: queryStringPtr(c, "payment_status"),
		BookingStatus: queryStringPtr(c, "booking_status"),
		Page:          page,
		PerPage:       perPage,
	}

	bookings, total, err := h.svc.ListBookings(c.Request().Context(), f)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch bookings")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": bookings, "total": total, "page": page, "per_page": perPage,
	})
}

// GetBooking godoc
//
//	@Summary     Get a booking with participants (admin)
//	@Tags        bookings
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id path string true "Booking ID"
//	@Success     200 {object} booking.Detail
//	@Failure     404 {object} map[string]interface{}
//	@Router      /api/v1/admin/bookings/{id} [get]
func (h *BookingHandler) GetBooking(c echo.Context) error {
	detail, err := h.svc.GetBooking(c.Request().Context(), c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch booking")
	}
	if detail == nil {
		return echo.NewHTTPError(http.StatusNotFound, "booking not found")
	}
	return c.JSON(http.StatusOK, detail)
}

// CreateBooking godoc
//
//	@Summary     Create a booking (admin)
//	@Tags        bookings
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Success     201 {object} map[string]interface{}
//	@Router      /api/v1/admin/bookings [post]
func (h *BookingHandler) CreateBooking(c echo.Context) error {
	var body struct {
		TourPackageID *string `json:"tour_package_id"`
		QuotationID   *string `json:"quotation_id"`
		CustomerName  string  `json:"customer_name"`
		CustomerEmail string  `json:"customer_email"`
		CustomerPhone string  `json:"customer_phone"`
		DepartureDate string  `json:"departure_date"`
		NumPeople     int     `json:"num_people"`
		TotalPrice    float64 `json:"total_price"`
		Notes         string  `json:"notes"`
		Participants  []struct {
			FullName    string `json:"full_name"`
			IDType      string `json:"id_type"`
			IDNumber    string `json:"id_number"`
			DateOfBirth string `json:"date_of_birth"`
			Phone       string `json:"phone"`
		} `json:"participants"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if body.CustomerName == "" || body.DepartureDate == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "customer_name and departure_date are required")
	}
	if body.NumPeople < 1 {
		body.NumPeople = 1
	}

	var participants []booking.ParticipantParams
	for _, p := range body.Participants {
		if p.FullName == "" || p.IDNumber == "" {
			continue
		}
		idType := p.IDType
		if idType == "" {
			idType = "ktp"
		}
		var dob, phone *string
		if p.DateOfBirth != "" {
			dob = &p.DateOfBirth
		}
		if p.Phone != "" {
			phone = &p.Phone
		}
		participants = append(participants, booking.ParticipantParams{
			FullName: p.FullName, IDType: idType, IDNumber: p.IDNumber,
			DateOfBirth: dob, Phone: phone,
		})
	}

	id, code, err := h.svc.CreateBooking(c.Request().Context(), booking.CreateParams{
		TourPackageID: body.TourPackageID,
		QuotationID:   body.QuotationID,
		CustomerName:  body.CustomerName,
		CustomerEmail: body.CustomerEmail,
		CustomerPhone: body.CustomerPhone,
		DepartureDate: body.DepartureDate,
		NumPeople:     body.NumPeople,
		TotalPrice:    body.TotalPrice,
		Notes:         body.Notes,
		Participants:  participants,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create booking")
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{"id": id, "booking_code": code})
}

// UpdatePaymentStatus godoc
//
//	@Summary     Update payment status (admin)
//	@Tags        bookings
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id   path string true "Booking ID"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/bookings/{id}/payment-status [patch]
func (h *BookingHandler) UpdatePaymentStatus(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		PaymentStatus string `json:"payment_status"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if !bookingsvc.ValidPaymentStatuses[body.PaymentStatus] {
		return echo.NewHTTPError(http.StatusBadRequest, "payment_status must be one of: pending, dp, lunas, refund")
	}

	err := h.svc.UpdatePaymentStatus(c.Request().Context(), id, body.PaymentStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "booking not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update payment status")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"id": id, "payment_status": body.PaymentStatus})
}

// UpdateBookingStatus godoc
//
//	@Summary     Update booking status (admin)
//	@Tags        bookings
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id   path string true "Booking ID"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/bookings/{id}/booking-status [patch]
func (h *BookingHandler) UpdateBookingStatus(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		BookingStatus string `json:"booking_status"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if !bookingsvc.ValidBookingStatuses[body.BookingStatus] {
		return echo.NewHTTPError(http.StatusBadRequest, "booking_status must be one of: confirmed, cancelled, completed")
	}

	err := h.svc.UpdateBookingStatus(c.Request().Context(), id, body.BookingStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "booking not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update booking status")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"id": id, "booking_status": body.BookingStatus})
}

// DeleteBooking godoc
//
//	@Summary     Delete a booking (admin)
//	@Tags        bookings
//	@Security    BearerAuth
//	@Param       id path string true "Booking ID"
//	@Success     204
//	@Router      /api/v1/admin/bookings/{id} [delete]
func (h *BookingHandler) DeleteBooking(c echo.Context) error {
	if err := h.svc.DeleteBooking(c.Request().Context(), c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete booking")
	}
	return c.NoContent(http.StatusNoContent)
}
