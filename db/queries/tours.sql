-- name: ListTourPackages :many
SELECT
    tp.*,
    d.name  AS destination_name,
    d.country AS destination_country
FROM tour_packages tp
LEFT JOIN destinations d ON d.id = tp.destination_id
WHERE tp.is_active = true
ORDER BY tp.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetTourPackageByID :one
SELECT
    tp.*,
    d.name  AS destination_name,
    d.country AS destination_country
FROM tour_packages tp
LEFT JOIN destinations d ON d.id = tp.destination_id
WHERE tp.id = $1;

-- name: GetTourPackageBySlug :one
SELECT
    tp.*,
    d.name  AS destination_name,
    d.country AS destination_country
FROM tour_packages tp
LEFT JOIN destinations d ON d.id = tp.destination_id
WHERE tp.slug = $1;

-- name: CreateTourPackage :one
INSERT INTO tour_packages (
    destination_id, title, slug, description, price, price_label,
    duration_days, max_participants, min_participants, package_type,
    cover_image_url, is_active
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: UpdateTourPackage :one
UPDATE tour_packages
SET
    destination_id   = $2,
    title            = $3,
    slug             = $4,
    description      = $5,
    price            = $6,
    price_label      = $7,
    duration_days    = $8,
    max_participants = $9,
    min_participants = $10,
    package_type     = $11,
    cover_image_url  = $12,
    is_active        = $13,
    updated_at       = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTourPackage :exec
DELETE FROM tour_packages WHERE id = $1;

-- name: CountTourPackages :one
SELECT COUNT(*) FROM tour_packages WHERE is_active = true;

-- name: ListItineraryByPackage :many
SELECT * FROM itinerary_items
WHERE tour_package_id = $1
ORDER BY day_number ASC, sort_order ASC;

-- name: CreateItineraryItem :one
INSERT INTO itinerary_items (
    tour_package_id, day_number, title, description,
    location, start_time, end_time, activity_type, sort_order
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: DeleteItineraryByPackage :exec
DELETE FROM itinerary_items WHERE tour_package_id = $1;

-- name: ListDestinations :many
SELECT * FROM destinations ORDER BY name ASC;

-- name: GetDestinationByID :one
SELECT * FROM destinations WHERE id = $1;

-- name: CreateDestination :one
INSERT INTO destinations (name, country, description, image_url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListPackageGalleries :many
SELECT * FROM package_galleries
WHERE tour_package_id = $1
ORDER BY sort_order ASC;

-- name: CreatePackageGallery :one
INSERT INTO package_galleries (tour_package_id, image_url, caption, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListPublishedTestimonials :many
SELECT * FROM testimonials
WHERE is_published = true
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListTestimonialsByPackage :many
SELECT * FROM testimonials
WHERE tour_package_id = $1 AND is_published = true
ORDER BY created_at DESC;

-- name: CreateTestimonial :one
INSERT INTO testimonials (tour_package_id, customer_name, content, rating, photo_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: PublishTestimonial :one
UPDATE testimonials SET is_published = true WHERE id = $1 RETURNING *;
