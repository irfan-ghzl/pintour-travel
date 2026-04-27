package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/booking"
)

// BookingRepo implements booking.Repository against PostgreSQL.
type BookingRepo struct {
	db *sql.DB
}

// NewBookingRepo creates a new BookingRepo.
func NewBookingRepo(db *sql.DB) *BookingRepo {
	return &BookingRepo{db: db}
}

func (r *BookingRepo) List(ctx context.Context, f booking.Filter) ([]booking.Booking, int, error) {
	args := []interface{}{}
	where := "1=1"
	argIdx := 0

	if f.PaymentStatus != nil {
		argIdx++
		where += fmt.Sprintf(" AND b.payment_status = $%d", argIdx)
		args = append(args, *f.PaymentStatus)
	}
	if f.BookingStatus != nil {
		argIdx++
		where += fmt.Sprintf(" AND b.booking_status = $%d", argIdx)
		args = append(args, *f.BookingStatus)
	}

	limitArg := argIdx + 1
	offsetArg := argIdx + 2
	args = append(args, f.PerPage, (f.Page-1)*f.PerPage)

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

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var bookings []booking.Booking
	for rows.Next() {
		var b booking.Booking
		if err := rows.Scan(
			&b.ID, &b.BookingCode, &b.CustomerName, &b.CustomerEmail, &b.CustomerPhone,
			&b.DepartureDate, &b.NumPeople, &b.TotalPrice,
			&b.PaymentStatus, &b.BookingStatus, &b.Notes, &b.CreatedAt, &b.PackageTitle,
		); err != nil {
			continue
		}
		bookings = append(bookings, b)
	}
	if bookings == nil {
		bookings = []booking.Booking{}
	}

	countArgs := args[:len(args)-2]
	var total int
	r.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM bookings b WHERE %s", where), countArgs...).Scan(&total) //nolint:errcheck

	return bookings, total, nil
}

func (r *BookingRepo) GetByID(ctx context.Context, id string) (*booking.Detail, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT b.id, b.booking_code, b.customer_name, b.customer_email, b.customer_phone,
		       b.departure_date, b.num_people, b.total_price,
		       b.payment_status, b.booking_status, b.notes, b.created_at,
		       tp.title AS package_title, tp.id AS package_id
		FROM bookings b
		LEFT JOIN tour_packages tp ON tp.id = b.tour_package_id
		WHERE b.id = $1`, id)

	var b booking.Booking
	var pkgTitle, pkgID *string
	if err := row.Scan(
		&b.ID, &b.BookingCode, &b.CustomerName, &b.CustomerEmail, &b.CustomerPhone,
		&b.DepartureDate, &b.NumPeople, &b.TotalPrice,
		&b.PaymentStatus, &b.BookingStatus, &b.Notes, &b.CreatedAt, &pkgTitle, &pkgID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	b.PackageTitle = pkgTitle
	b.TourPackageID = pkgID

	pRows, err := r.db.QueryContext(ctx, `
		SELECT id, full_name, id_type, id_number, date_of_birth, phone
		FROM booking_participants
		WHERE booking_id = $1
		ORDER BY created_at ASC`, b.ID)

	var participants []booking.Participant
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var p booking.Participant
			if err := pRows.Scan(&p.ID, &p.FullName, &p.IDType, &p.IDNumber, &p.DateOfBirth, &p.Phone); err != nil {
				continue
			}
			participants = append(participants, p)
		}
	}
	if participants == nil {
		participants = []booking.Participant{}
	}

	return &booking.Detail{Booking: b, Participants: participants}, nil
}

func (r *BookingRepo) Create(ctx context.Context, p booking.CreateParams) (id, code string, err error) {
	code = fmt.Sprintf("BK-%s-%04d", time.Now().Format("20060102"), time.Now().UnixMilli()%10000)

	tx, txErr := r.db.BeginTx(ctx, nil)
	if txErr != nil {
		return "", "", txErr
	}
	defer tx.Rollback() //nolint:errcheck

	var quotationID *string
	if p.QuotationID != nil && *p.QuotationID != "" {
		quotationID = p.QuotationID
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO bookings
		  (tour_package_id, quotation_id, booking_code, customer_name, customer_email,
		   customer_phone, departure_date, num_people, total_price, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id`,
		p.TourPackageID, quotationID, code,
		p.CustomerName, nullString(p.CustomerEmail), nullString(p.CustomerPhone),
		p.DepartureDate, p.NumPeople, p.TotalPrice, nullString(p.Notes),
	).Scan(&id)
	if err != nil {
		return "", "", err
	}

	for _, part := range p.Participants {
		if part.FullName == "" || part.IDNumber == "" {
			continue
		}
		idType := part.IDType
		if idType == "" {
			idType = "ktp"
		}
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO booking_participants (booking_id, full_name, id_type, id_number, date_of_birth, phone)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			id, part.FullName, idType, part.IDNumber, part.DateOfBirth, part.Phone,
		); err != nil {
			return "", "", err
		}
	}

	if err = tx.Commit(); err != nil {
		return "", "", err
	}
	return id, code, nil
}

func (r *BookingRepo) UpdatePaymentStatus(ctx context.Context, id, status string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE bookings SET payment_status=$2, updated_at=NOW() WHERE id=$1`, id, status)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *BookingRepo) UpdateBookingStatus(ctx context.Context, id, status string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE bookings SET booking_status=$2, updated_at=NOW() WHERE id=$1`, id, status)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *BookingRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM bookings WHERE id=$1`, id)
	return err
}

// nullString returns nil if s is empty, otherwise a pointer to s.
func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
