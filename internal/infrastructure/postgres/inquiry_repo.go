package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/inquiry"
)

// InquiryRepo implements inquiry.Repository against PostgreSQL.
type InquiryRepo struct {
	db *sql.DB
}

// NewInquiryRepo creates a new InquiryRepo.
func NewInquiryRepo(db *sql.DB) *InquiryRepo {
	return &InquiryRepo{db: db}
}

func (r *InquiryRepo) Create(ctx context.Context, p inquiry.CreateParams) (id, createdAt string, err error) {
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO inquiries
		  (full_name, email, phone, destination, tour_package_id,
		   num_people, budget, duration_days, departure_date, notes, wa_link)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, created_at`,
		p.FullName, nullString(p.Email), nullString(p.Phone),
		nullString(p.Destination), p.TourPackageID,
		p.NumPeople, nullFloat(p.Budget), nullInt(p.DurationDays),
		p.DepartureDate, nullString(p.Notes), p.WALink,
	).Scan(&id, &createdAt)
	return
}

func (r *InquiryRepo) List(ctx context.Context, f inquiry.Filter) ([]inquiry.Inquiry, int, error) {
	args := []interface{}{}
	where := "1=1"
	argIdx := 0

	if f.Status != "" {
		argIdx++
		where += fmt.Sprintf(" AND i.status = $%d", argIdx)
		args = append(args, f.Status)
	}

	limitArg := argIdx + 1
	offsetArg := argIdx + 2
	args = append(args, f.PerPage, (f.Page-1)*f.PerPage)

	query := fmt.Sprintf(`
		SELECT i.id, i.full_name, i.email, i.phone, i.destination,
		       i.num_people, i.budget, i.duration_days, i.departure_date,
		       i.status, i.wa_link, i.created_at,
		       COALESCE(tp.title, '') AS package_title
		FROM inquiries i
		LEFT JOIN tour_packages tp ON tp.id = i.tour_package_id
		WHERE %s
		ORDER BY i.created_at DESC
		LIMIT $%d OFFSET $%d`, where, limitArg, offsetArg)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var inquiries []inquiry.Inquiry
	for rows.Next() {
		var inq inquiry.Inquiry
		if err := rows.Scan(
			&inq.ID, &inq.FullName, &inq.Email, &inq.Phone, &inq.Destination,
			&inq.NumPeople, &inq.Budget, &inq.DurationDays, &inq.DepartureDate,
			&inq.Status, &inq.WALink, &inq.CreatedAt, &inq.PackageTitle,
		); err != nil {
			continue
		}
		inquiries = append(inquiries, inq)
	}
	if inquiries == nil {
		inquiries = []inquiry.Inquiry{}
	}

	countArgs := args[:len(args)-2]
	var total int
	r.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM inquiries i WHERE %s", where), countArgs...).Scan(&total) //nolint:errcheck

	return inquiries, total, nil
}

func (r *InquiryRepo) UpdateStatus(ctx context.Context, id, status string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE inquiries SET status=$2, updated_at=NOW() WHERE id=$1`, id, status)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *InquiryRepo) GetPackageTitle(ctx context.Context, packageID string) (string, error) {
	var title string
	err := r.db.QueryRowContext(ctx,
		`SELECT title FROM tour_packages WHERE id = $1`, packageID,
	).Scan(&title)
	if err != nil {
		return "", err
	}
	return title, nil
}

func nullFloat(f float64) interface{} {
	if f == 0 {
		return nil
	}
	return f
}

func nullInt(i int) interface{} {
	if i == 0 {
		return nil
	}
	return i
}
