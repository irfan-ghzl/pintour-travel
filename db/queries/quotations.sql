-- name: CreateQuotation :one
INSERT INTO quotations (
    inquiry_id, created_by, title, customer_name,
    customer_email, customer_phone, valid_until, total_price, notes
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetQuotationByID :one
SELECT q.*,
       i.full_name AS inquiry_customer_name
FROM quotations q
LEFT JOIN inquiries i ON i.id = q.inquiry_id
WHERE q.id = $1;

-- name: ListQuotations :many
SELECT q.*,
       u.name AS created_by_name
FROM quotations q
LEFT JOIN users u ON u.id = q.created_by
ORDER BY q.created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateQuotationStatus :one
UPDATE quotations
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateQuotationPDFUrl :one
UPDATE quotations
SET pdf_url = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateQuotation :one
UPDATE quotations
SET
    title          = $2,
    customer_name  = $3,
    customer_email = $4,
    customer_phone = $5,
    valid_until    = $6,
    total_price    = $7,
    notes          = $8,
    status         = $9,
    updated_at     = NOW()
WHERE id = $1
RETURNING *;

-- name: AddQuotationItem :one
INSERT INTO quotation_items (quotation_id, description, category, quantity, unit_price)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListQuotationItems :many
SELECT * FROM quotation_items WHERE quotation_id = $1;

-- name: DeleteQuotationItems :exec
DELETE FROM quotation_items WHERE quotation_id = $1;
