-- name: CreateInquiry :one
INSERT INTO inquiries (
    full_name, email, phone, destination, tour_package_id,
    num_people, budget, duration_days, departure_date, notes, wa_link
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListInquiries :many
SELECT i.*,
       tp.title AS package_title,
       u.name   AS assigned_name
FROM inquiries i
LEFT JOIN tour_packages tp ON tp.id = i.tour_package_id
LEFT JOIN users u ON u.id = i.assigned_to
ORDER BY i.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetInquiryByID :one
SELECT i.*,
       tp.title AS package_title,
       u.name   AS assigned_name
FROM inquiries i
LEFT JOIN tour_packages tp ON tp.id = i.tour_package_id
LEFT JOIN users u ON u.id = i.assigned_to
WHERE i.id = $1;

-- name: UpdateInquiryStatus :one
UPDATE inquiries
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: AssignInquiry :one
UPDATE inquiries
SET assigned_to = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CountInquiriesByStatus :many
SELECT status, COUNT(*) AS total
FROM inquiries
GROUP BY status;

-- name: CountNewInquiries :one
SELECT COUNT(*) FROM inquiries WHERE status = 'new';
