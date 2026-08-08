package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/invoice"
)

type invoiceRepo struct{ db *sql.DB }

func NewInvoiceRepo(db *sql.DB) invoice.Repository { return &invoiceRepo{db} }

func (r *invoiceRepo) Create(ctx context.Context, inv *invoice.Invoice) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO invoices
		(id,invoice_number,participant_id,batch_id,amount,due_date,status,
		 pdf_path,notes,issued_by,created_at,updated_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())
		RETURNING id,created_at,updated_at`,
		inv.InvoiceNumber, inv.ParticipantID, inv.BatchID, inv.Amount, inv.DueDate,
		inv.Status, inv.PDFPath, inv.Notes, inv.IssuedBy,
	).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
}

func (r *invoiceRepo) Update(ctx context.Context, inv *invoice.Invoice) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE invoices SET amount=$1,due_date=$2,status=$3,pdf_path=$4,notes=$5,updated_at=NOW()
		WHERE id=$6`,
		inv.Amount, inv.DueDate, inv.Status, inv.PDFPath, inv.Notes, inv.ID)
	return err
}

func (r *invoiceRepo) GetByID(ctx context.Context, id string) (*invoice.Invoice, error) {
	return r.scan(ctx, `
		SELECT i.id,i.invoice_number,i.participant_id,i.batch_id,i.amount,i.due_date,i.status,
		COALESCE(i.pdf_path,''),COALESCE(i.notes,''),i.issued_by,i.confirmed_by,i.confirmed_at,
		i.created_at,i.updated_at,
		p.name,p.phone,pkg.name,u.name
		FROM invoices i
		JOIN participants p ON p.id=i.participant_id
		JOIN package_batches pb ON pb.id=i.batch_id
		JOIN packages pkg ON pkg.id=pb.package_id
		JOIN users u ON u.id=i.issued_by
		WHERE i.id=$1`, id)
}

func (r *invoiceRepo) GetByNumber(ctx context.Context, number string) (*invoice.Invoice, error) {
	return r.scan(ctx, `
		SELECT i.id,i.invoice_number,i.participant_id,i.batch_id,i.amount,i.due_date,i.status,
		COALESCE(i.pdf_path,''),COALESCE(i.notes,''),i.issued_by,i.confirmed_by,i.confirmed_at,
		i.created_at,i.updated_at,
		p.name,p.phone,pkg.name,u.name
		FROM invoices i
		JOIN participants p ON p.id=i.participant_id
		JOIN package_batches pb ON pb.id=i.batch_id
		JOIN packages pkg ON pkg.id=pb.package_id
		JOIN users u ON u.id=i.issued_by
		WHERE i.invoice_number=$1`, number)
}

func (r *invoiceRepo) scan(ctx context.Context, q string, arg interface{}) (*invoice.Invoice, error) {
	var inv invoice.Invoice
	err := r.db.QueryRowContext(ctx, q, arg).Scan(
		&inv.ID, &inv.InvoiceNumber, &inv.ParticipantID, &inv.BatchID,
		&inv.Amount, &inv.DueDate, &inv.Status, &inv.PDFPath, &inv.Notes,
		&inv.IssuedBy, &inv.ConfirmedBy, &inv.ConfirmedAt,
		&inv.CreatedAt, &inv.UpdatedAt,
		&inv.ParticipantName, &inv.ParticipantPhone, &inv.PackageName, &inv.IssuedByName)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *invoiceRepo) List(ctx context.Context, f invoice.Filter) ([]invoice.Invoice, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	i := 1

	if f.Status != nil {
		where += fmt.Sprintf(" AND i.status=$%d", i)
		args = append(args, *f.Status)
		i++
	}
	if f.ParticipantID != nil {
		where += fmt.Sprintf(" AND i.participant_id=$%d", i)
		args = append(args, *f.ParticipantID)
		i++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM invoices i
		JOIN participants p ON p.id=i.participant_id
		JOIN package_batches pb ON pb.id=i.batch_id
		JOIN packages pkg ON pkg.id=pb.package_id
		JOIN users u ON u.id=i.issued_by `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = 20
	}
	offset := (f.Page - 1) * f.PerPage
	args = append(args, f.PerPage, offset)

	q := fmt.Sprintf(`
		SELECT i.id,i.invoice_number,i.participant_id,i.batch_id,i.amount,i.due_date,i.status,
		COALESCE(i.pdf_path,''),COALESCE(i.notes,''),i.issued_by,i.confirmed_by,i.confirmed_at,
		i.created_at,i.updated_at,
		p.name,p.phone,pkg.name,u.name
		FROM invoices i
		JOIN participants p ON p.id=i.participant_id
		JOIN package_batches pb ON pb.id=i.batch_id
		JOIN packages pkg ON pkg.id=pb.package_id
		JOIN users u ON u.id=i.issued_by
		%s ORDER BY i.created_at DESC LIMIT $%d OFFSET $%d`, where, i, i+1)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []invoice.Invoice
	for rows.Next() {
		var inv invoice.Invoice
		if err := rows.Scan(
			&inv.ID, &inv.InvoiceNumber, &inv.ParticipantID, &inv.BatchID,
			&inv.Amount, &inv.DueDate, &inv.Status, &inv.PDFPath, &inv.Notes,
			&inv.IssuedBy, &inv.ConfirmedBy, &inv.ConfirmedAt,
			&inv.CreatedAt, &inv.UpdatedAt,
			&inv.ParticipantName, &inv.ParticipantPhone, &inv.PackageName, &inv.IssuedByName); err != nil {
			return nil, 0, err
		}
		list = append(list, inv)
	}
	return list, total, rows.Err()
}

func (r *invoiceRepo) Confirm(ctx context.Context, id, confirmedBy string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE invoices SET status='lunas',confirmed_by=$1,confirmed_at=NOW(),updated_at=NOW()
		WHERE id=$2`, confirmedBy, id)
	return err
}

// NextSequence returns the next sequence number for an invoice in the given month.
// Uses MAX(seq)+1 to be resilient to soft-deleted rows (§13.7 — sequence must
// be unique per month, soft-delete tidak boleh re-use nomor).
func (r *invoiceRepo) NextSequence(ctx context.Context, yearMonth string) (int, error) {
	var maxSeq int
	prefix := "INV-" + yearMonth + "-"
	// Extract the last 4 digits, take MAX, +1. Includes soft-deleted to avoid number reuse.
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(CAST(SUBSTRING(invoice_number FROM LENGTH($2)+1) AS INTEGER)), 0)
		FROM invoices
		WHERE invoice_number LIKE $1`,
		prefix+"%", prefix).Scan(&maxSeq)
	if err != nil {
		return 0, err
	}
	return maxSeq + 1, nil
}

func (r *invoiceRepo) ListUnpaidOlderThan(ctx context.Context, days int) ([]invoice.Invoice, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.id,i.invoice_number,i.participant_id,i.batch_id,i.amount,i.due_date,i.status,
		COALESCE(i.pdf_path,''),COALESCE(i.notes,''),i.issued_by,i.confirmed_by,i.confirmed_at,
		i.created_at,i.updated_at,
		p.name,p.phone,pkg.name,u.name
		FROM invoices i
		JOIN participants p ON p.id=i.participant_id
		JOIN package_batches pb ON pb.id=i.batch_id
		JOIN packages pkg ON pkg.id=pb.package_id
		JOIN users u ON u.id=i.issued_by
		WHERE i.status IN ('diterbitkan','menunggu_bayar')
		AND i.created_at < NOW() - ($1 || ' days')::interval
		ORDER BY i.created_at`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []invoice.Invoice
	for rows.Next() {
		var inv invoice.Invoice
		if err := rows.Scan(
			&inv.ID, &inv.InvoiceNumber, &inv.ParticipantID, &inv.BatchID,
			&inv.Amount, &inv.DueDate, &inv.Status, &inv.PDFPath, &inv.Notes,
			&inv.IssuedBy, &inv.ConfirmedBy, &inv.ConfirmedAt,
			&inv.CreatedAt, &inv.UpdatedAt,
			&inv.ParticipantName, &inv.ParticipantPhone, &inv.PackageName, &inv.IssuedByName); err != nil {
			return nil, err
		}
		list = append(list, inv)
	}
	return list, rows.Err()
}

// ─── PaymentProof ─────────────────────────────────────────────────────────────

type paymentProofRepo struct{ db *sql.DB }

func NewPaymentProofRepo(db *sql.DB) invoice.PaymentProofRepository {
	return &paymentProofRepo{db}
}

func (r *paymentProofRepo) Create(ctx context.Context, pp *invoice.PaymentProof) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO payment_proofs
		(id,invoice_id,file_path,amount_claimed,notes,status,uploaded_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,'menunggu',NOW())
		RETURNING id,uploaded_at`,
		pp.InvoiceID, pp.FilePath, pp.AmountClaimed, pp.Notes,
	).Scan(&pp.ID, &pp.UploadedAt)
}

// proofCols is the column list both proof queries select, in the order
// scanProof reads them.
const proofCols = `id,invoice_id,file_path,amount_claimed,COALESCE(notes,''),
	status,reviewed_by,COALESCE(review_notes,''),uploaded_at,reviewed_at`

// proofScanner is satisfied by both *sql.Row and *sql.Rows, so one scan serves
// the single-row and the multi-row query.
type proofScanner interface{ Scan(dest ...any) error }

func scanProof(src proofScanner, pp *invoice.PaymentProof) error {
	return src.Scan(&pp.ID, &pp.InvoiceID, &pp.FilePath, &pp.AmountClaimed,
		&pp.Notes, &pp.Status, &pp.ReviewedBy, &pp.ReviewNotes,
		&pp.UploadedAt, &pp.ReviewedAt)
}

func (r *paymentProofRepo) GetByID(ctx context.Context, id string) (*invoice.PaymentProof, error) {
	var pp invoice.PaymentProof
	row := r.db.QueryRowContext(ctx,
		`SELECT `+proofCols+` FROM payment_proofs WHERE id=$1`, id)
	if err := scanProof(row, &pp); err != nil {
		return nil, err
	}
	return &pp, nil
}

func (r *paymentProofRepo) GetByInvoice(ctx context.Context, invoiceID string) ([]invoice.PaymentProof, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+proofCols+` FROM payment_proofs WHERE invoice_id=$1 ORDER BY uploaded_at`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []invoice.PaymentProof
	for rows.Next() {
		var pp invoice.PaymentProof
		if err := scanProof(rows, &pp); err != nil {
			return nil, err
		}
		list = append(list, pp)
	}
	return list, rows.Err()
}

func (r *paymentProofRepo) Review(ctx context.Context, id, status, reviewedBy, notes string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE payment_proofs SET status=$1,reviewed_by=$2,review_notes=$3,reviewed_at=$4
		WHERE id=$5`, status, reviewedBy, notes, now, id)
	return err
}
