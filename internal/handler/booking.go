package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// BookingHandler handles HTTP requests for bookings and participants.
type BookingHandler struct {
	db *sql.DB
}

// NewBookingHandler creates a new BookingHandler.
func NewBookingHandler(db *sql.DB) *BookingHandler {
	return &BookingHandler{db: db}
}

// ListBookings godoc
//
//	@Summary     List bookings (admin)
//	@Tags        bookings
//	@Produce     json
//	@Security    BearerAuth
//	@Param       page           query int    false "Page (default 1)"
//	@Param       per_page       query int    false "Per page (default 15)"
//	@Param       payment_status query string false "Filter by payment_status"
//	@Param       booking_status query string false "Filter by booking_status"
//	@Success     200 {object} map[string]interface{}
//	@Router      /api/v1/admin/bookings [get]
func (h *BookingHandler) ListBookings(c echo.Context) error {
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 15)
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	args := []interface{}{}
	where := "1=1"
	argIdx := 0

	if ps := c.QueryParam("payment_status"); ps != "" {
		argIdx++
		where += fmt.Sprintf(" AND b.payment_status = $%d", argIdx)
		args = append(args, ps)
	}
	if bs := c.QueryParam("booking_status"); bs != "" {
		argIdx++
		where += fmt.Sprintf(" AND b.booking_status = $%d", argIdx)
		args = append(args, bs)
	}

	limitArg := argIdx + 1
	offsetArg := argIdx + 2
	args = append(args, perPage, offset)

	query := fmt.Sprintf(`
		SELECT b.id, b.booking_code, b.customer_name, b.customer_email, b.customer_phone,
		       b.departure_date, b.num_people, b.total_price,
		       b.payment_status, b.booking_status, b.notes, b.created_at,
		       tp.title AS package_title
		FROM bookings b
		LEFT JOIN tour_packages tp ON tp.id = b.tour_package_id
		WHERE %s
		ORDER BY b.created_at DESC
		LIMIT $%d OFFSET $%d`, where, limitArg, offsetArg)

	rows, err := h.db.QueryContext(c.Request().Context(), query, args...)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch bookings")
	}
	defer rows.Close()

	var bookings []map[string]interface{}
	for rows.Next() {
		var (
			id, code, name, payStatus, bookStatus, createdAt string
			depDate                                          string
			numPeople                                        int
			totalPrice                                       float64
			email, phone, notes, pkgTitle                    *string
		)
		if err := rows.Scan(&id, &code, &name, &email, &phone,
			&depDate, &numPeople, &totalPrice,
			&payStatus, &bookStatus, &notes, &createdAt, &pkgTitle); err != nil {
			continue
		}
		bookings = append(bookings, map[string]interface{}{
			"id": id, "booking_code": code, "customer_name": name,
			"customer_email": email, "customer_phone": phone,
			"departure_date": depDate, "num_people": numPeople,
			"total_price": totalPrice, "payment_status": payStatus,
			"booking_status": bookStatus, "notes": notes,
			"created_at": createdAt, "package_title": pkgTitle,
		})
	}
	if bookings == nil {
		bookings = []map[string]interface{}{}
	}

	countArgs := args[:len(args)-2]
	var total int
	h.db.QueryRowContext(c.Request().Context(),
		fmt.Sprintf(`SELECT COUNT(*) FROM bookings b WHERE %s`, where), countArgs...).Scan(&total)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":     bookings,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// GetBooking godoc
//
//	@Summary     Get a booking with participants (admin)
//	@Tags        bookings
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id path string true "Booking ID"
//	@Success     200 {object} map[string]interface{}
//	@Failure     404 {object} map[string]interface{}
//	@Router      /api/v1/admin/bookings/{id} [get]
func (h *BookingHandler) GetBooking(c echo.Context) error {
	id := c.Param("id")

	row := h.db.QueryRowContext(c.Request().Context(), `
		SELECT b.id, b.booking_code, b.customer_name, b.customer_email, b.customer_phone,
		       b.departure_date, b.num_people, b.total_price,
		       b.payment_status, b.booking_status, b.notes, b.created_at,
		       tp.title AS package_title, tp.id AS package_id
		FROM bookings b
		LEFT JOIN tour_packages tp ON tp.id = b.tour_package_id
		WHERE b.id = $1`, id)

	var (
		bID, code, name, payStatus, bookStatus, createdAt string
		depDate                                           string
		numPeople                                         int
		totalPrice                                        float64
		email, phone, notes, pkgTitle, pkgID              *string
	)
	if err := row.Scan(&bID, &code, &name, &email, &phone,
		&depDate, &numPeople, &totalPrice,
		&payStatus, &bookStatus, &notes, &createdAt, &pkgTitle, &pkgID); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "booking not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch booking")
	}

	booking := map[string]interface{}{
		"id": bID, "booking_code": code, "customer_name": name,
		"customer_email": email, "customer_phone": phone,
		"departure_date": depDate, "num_people": numPeople,
		"total_price": totalPrice, "payment_status": payStatus,
		"booking_status": bookStatus, "notes": notes,
		"created_at": createdAt, "package_title": pkgTitle, "package_id": pkgID,
	}

	// Fetch participants
	pRows, err := h.db.QueryContext(c.Request().Context(), `
		SELECT id, full_name, id_type, id_number, date_of_birth, phone
		FROM booking_participants
		WHERE booking_id = $1
		ORDER BY created_at ASC`, bID)
	if err == nil {
		defer pRows.Close()
		var participants []map[string]interface{}
		for pRows.Next() {
			var pID, fullName, idType, idNumber string
			var dob, pPhone *string
			if err := pRows.Scan(&pID, &fullName, &idType, &idNumber, &dob, &pPhone); err != nil {
				continue
			}
			participants = append(participants, map[string]interface{}{
				"id": pID, "full_name": fullName,
				"id_type": idType, "id_number": idNumber,
				"date_of_birth": dob, "phone": pPhone,
			})
		}
		if participants == nil {
			participants = []map[string]interface{}{}
		}
		booking["participants"] = participants
	}

	return c.JSON(http.StatusOK, booking)
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

	// Generate booking code: BK-YYYYMMDD-XXXX
	code := fmt.Sprintf("BK-%s-%04d", time.Now().Format("20060102"), time.Now().UnixMilli()%10000)

	tx, err := h.db.BeginTx(c.Request().Context(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to start transaction")
	}
	defer tx.Rollback() //nolint:errcheck

	var bookingID string
	err = tx.QueryRowContext(c.Request().Context(), `
		INSERT INTO bookings
		  (tour_package_id, quotation_id, booking_code, customer_name, customer_email,
		   customer_phone, departure_date, num_people, total_price, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id`,
		body.TourPackageID, body.QuotationID, code,
		body.CustomerName, body.CustomerEmail, body.CustomerPhone,
		body.DepartureDate, body.NumPeople, body.TotalPrice, body.Notes,
	).Scan(&bookingID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create booking")
	}

	for _, p := range body.Participants {
		if p.FullName == "" || p.IDNumber == "" {
			continue
		}
		idType := p.IDType
		if idType == "" {
			idType = "ktp"
		}
		var dob *string
		if p.DateOfBirth != "" {
			dob = &p.DateOfBirth
		}
		var phone *string
		if p.Phone != "" {
			phone = &p.Phone
		}
		if _, err := tx.ExecContext(c.Request().Context(), `
			INSERT INTO booking_participants (booking_id, full_name, id_type, id_number, date_of_birth, phone)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			bookingID, p.FullName, idType, p.IDNumber, dob, phone,
		); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to add participant")
		}
	}

	if err := tx.Commit(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to commit transaction")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":           bookingID,
		"booking_code": code,
	})
}

// UpdatePaymentStatus godoc
//
//	@Summary     Update payment status (admin)
//	@Tags        bookings
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id   path string true "Booking ID"
//	@Param       body body object true "payment_status: pending|dp|lunas|refund"
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

	valid := map[string]bool{"pending": true, "dp": true, "lunas": true, "refund": true}
	if !valid[body.PaymentStatus] {
		return echo.NewHTTPError(http.StatusBadRequest, "payment_status must be one of: pending, dp, lunas, refund")
	}

	res, err := h.db.ExecContext(c.Request().Context(),
		`UPDATE bookings SET payment_status=$2, updated_at=NOW() WHERE id=$1`, id, body.PaymentStatus)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update payment status")
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "booking not found")
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
//	@Param       body body object true "booking_status: confirmed|cancelled|completed"
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

	valid := map[string]bool{"confirmed": true, "cancelled": true, "completed": true}
	if !valid[body.BookingStatus] {
		return echo.NewHTTPError(http.StatusBadRequest, "booking_status must be one of: confirmed, cancelled, completed")
	}

	res, err := h.db.ExecContext(c.Request().Context(),
		`UPDATE bookings SET booking_status=$2, updated_at=NOW() WHERE id=$1`, id, body.BookingStatus)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update booking status")
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "booking not found")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"id": id, "booking_status": body.BookingStatus})
}

// DeleteBooking godoc
//
//	@Summary     Delete a booking (admin)
//	@Tags        bookings
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id path string true "Booking ID"
//	@Success     204
//	@Router      /api/v1/admin/bookings/{id} [delete]
func (h *BookingHandler) DeleteBooking(c echo.Context) error {
	id := c.Param("id")
	if _, err := h.db.ExecContext(c.Request().Context(),
		`DELETE FROM bookings WHERE id=$1`, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete booking")
	}
	return c.NoContent(http.StatusNoContent)
}
