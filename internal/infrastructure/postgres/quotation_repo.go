package postgres

import (
	"context"
	"database/sql"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/quotation"
)

// QuotationRepo implements quotation.Repository against PostgreSQL.
type QuotationRepo struct {
	db *sql.DB
}

// NewQuotationRepo creates a new QuotationRepo.
func NewQuotationRepo(db *sql.DB) *QuotationRepo {
	return &QuotationRepo{db: db}
}

func (r *QuotationRepo) Create(ctx context.Context, p quotation.CreateParams) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback() //nolint:errcheck

	var id string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO quotations
		  (inquiry_id, created_by, title, customer_name, customer_email,
		   customer_phone, valid_until, total_price, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id`,
		p.InquiryID, p.CreatedBy, p.Title, p.CustomerName,
		nullString(p.CustomerEmail), nullString(p.CustomerPhone),
		p.ValidUntil, p.TotalPrice, nullString(p.Notes),
	).Scan(&id)
	if err != nil {
		return "", err
	}

	for _, item := range p.Items {
		qty := item.Quantity
		if qty <= 0 {
			qty = 1
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO quotation_items (quotation_id, description, category, quantity, unit_price)
			VALUES ($1,$2,$3,$4,$5)`,
			id, item.Description, nullString(item.Category), qty, item.UnitPrice,
		); err != nil {
			return "", err
		}
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return id, nil
}

func (r *QuotationRepo) GetByID(ctx context.Context, id string) (*quotation.Detail, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, customer_name, customer_email, customer_phone,
		       valid_until, total_price, notes, status, pdf_url, created_at
		FROM quotations WHERE id=$1`, id)

	var q quotation.Quotation
	if err := row.Scan(
		&q.ID, &q.Title, &q.CustomerName, &q.CustomerEmail, &q.CustomerPhone,
		&q.ValidUntil, &q.TotalPrice, &q.Notes, &q.Status, &q.PDFURL, &q.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	itemRows, err := r.db.QueryContext(ctx, `
		SELECT id, description, category, quantity, unit_price, total_price
		FROM quotation_items WHERE quotation_id=$1`, id)
	if err != nil {
		return &quotation.Detail{Quotation: q, Items: []quotation.Item{}}, nil
	}
	defer itemRows.Close()

	var items []quotation.Item
	for itemRows.Next() {
		var it quotation.Item
		if err := itemRows.Scan(&it.ID, &it.Description, &it.Category, &it.Quantity, &it.UnitPrice, &it.TotalPrice); err != nil {
			continue
		}
		items = append(items, it)
	}
	if items == nil {
		items = []quotation.Item{}
	}

	return &quotation.Detail{Quotation: q, Items: items}, nil
}

func (r *QuotationRepo) List(ctx context.Context, f quotation.Filter) ([]quotation.Quotation, int, error) {
	offset := (f.Page - 1) * f.PerPage
	rows, err := r.db.QueryContext(ctx, `
		SELECT q.id, q.title, q.customer_name, q.total_price,
		       q.status, q.created_at, COALESCE(u.name,'') AS created_by_name
		FROM quotations q
		LEFT JOIN users u ON u.id = q.created_by
		ORDER BY q.created_at DESC
		LIMIT $1 OFFSET $2`, f.PerPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var quotations []quotation.Quotation
	for rows.Next() {
		var q quotation.Quotation
		if err := rows.Scan(&q.ID, &q.Title, &q.CustomerName, &q.TotalPrice,
			&q.Status, &q.CreatedAt, &q.CreatedByName); err != nil {
			continue
		}
		quotations = append(quotations, q)
	}
	if quotations == nil {
		quotations = []quotation.Quotation{}
	}

	var total int
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM quotations`).Scan(&total) //nolint:errcheck

	return quotations, total, nil
}
