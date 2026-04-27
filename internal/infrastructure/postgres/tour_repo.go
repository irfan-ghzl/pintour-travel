package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/tour"
)

// TourRepo implements tour.Repository against PostgreSQL.
type TourRepo struct {
	db *sql.DB
}

// NewTourRepo creates a new TourRepo.
func NewTourRepo(db *sql.DB) *TourRepo {
	return &TourRepo{db: db}
}

func (r *TourRepo) List(ctx context.Context, f tour.Filter) ([]tour.TourPackage, int, error) {
	args := []interface{}{}
	where := "tp.is_active = true"
	argIdx := 0

	if f.DestinationID != nil {
		argIdx++
		where += " AND tp.destination_id = $" + strconv.Itoa(argIdx)
		args = append(args, *f.DestinationID)
	}
	if f.PackageType != nil {
		argIdx++
		where += " AND tp.package_type = $" + strconv.Itoa(argIdx)
		args = append(args, *f.PackageType)
	}
	if f.PriceMin != nil {
		argIdx++
		where += " AND tp.price >= $" + strconv.Itoa(argIdx)
		args = append(args, *f.PriceMin)
	}
	if f.PriceMax != nil {
		argIdx++
		where += " AND tp.price <= $" + strconv.Itoa(argIdx)
		args = append(args, *f.PriceMax)
	}
	if f.DurationDays != nil {
		argIdx++
		where += " AND tp.duration_days = $" + strconv.Itoa(argIdx)
		args = append(args, *f.DurationDays)
	}

	limitArg := argIdx + 1
	offsetArg := argIdx + 2
	args = append(args, f.PerPage, (f.Page-1)*f.PerPage)

	query := fmt.Sprintf(`
		SELECT tp.id, tp.title, tp.slug, tp.description, tp.price, tp.price_label,
		       tp.duration_days, tp.max_participants, tp.min_participants,
		       tp.package_type, tp.cover_image_url, tp.is_active, tp.created_at,
		       d.name, d.country
		FROM tour_packages tp
		LEFT JOIN destinations d ON d.id = tp.destination_id
		WHERE %s
		ORDER BY tp.created_at DESC
		LIMIT $%d OFFSET $%d`, where, limitArg, offsetArg)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var packages []tour.TourPackage
	for rows.Next() {
		var p tour.TourPackage
		if err := rows.Scan(
			&p.ID, &p.Title, &p.Slug, &p.Description, &p.Price, &p.PriceLabel,
			&p.DurationDays, &p.MaxParticipants, &p.MinParticipants,
			&p.PackageType, &p.CoverImageURL, &p.IsActive, &p.CreatedAt,
			&p.DestinationName, &p.DestinationCountry,
		); err != nil {
			continue
		}
		packages = append(packages, p)
	}
	if packages == nil {
		packages = []tour.TourPackage{}
	}

	countArgs := args[:len(args)-2]
	var total int
	r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tour_packages tp WHERE "+where, countArgs...).Scan(&total) //nolint:errcheck

	return packages, total, nil
}

func (r *TourRepo) GetBySlug(ctx context.Context, slug string) (*tour.PackageDetail, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tp.id, tp.title, tp.slug, tp.description, tp.price, tp.price_label,
		       tp.duration_days, tp.max_participants, tp.min_participants,
		       tp.package_type, tp.cover_image_url, tp.is_active, tp.created_at,
		       COALESCE(d.name,'') AS destination_name,
		       COALESCE(d.country,'') AS destination_country
		FROM tour_packages tp
		LEFT JOIN destinations d ON d.id = tp.destination_id
		WHERE tp.slug = $1`, slug)

	var p tour.TourPackage
	var destName, destCountry string
	if err := row.Scan(
		&p.ID, &p.Title, &p.Slug, &p.Description, &p.Price, &p.PriceLabel,
		&p.DurationDays, &p.MaxParticipants, &p.MinParticipants,
		&p.PackageType, &p.CoverImageURL, &p.IsActive, &p.CreatedAt,
		&destName, &destCountry,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p.DestinationName = &destName
	p.DestinationCountry = &destCountry

	itRows, err := r.db.QueryContext(ctx, `
		SELECT id, day_number, title, description, location, start_time, end_time,
		       activity_type, sort_order
		FROM itinerary_items
		WHERE tour_package_id = $1
		ORDER BY day_number ASC, sort_order ASC`, p.ID)
	if err != nil {
		return &tour.PackageDetail{TourPackage: p, Itinerary: []tour.ItineraryItem{}}, nil
	}
	defer itRows.Close()

	var items []tour.ItineraryItem
	for itRows.Next() {
		var it tour.ItineraryItem
		if err := itRows.Scan(&it.ID, &it.DayNumber, &it.Title, &it.Description,
			&it.Location, &it.StartTime, &it.EndTime, &it.ActivityType, &it.SortOrder); err != nil {
			continue
		}
		items = append(items, it)
	}
	if items == nil {
		items = []tour.ItineraryItem{}
	}

	return &tour.PackageDetail{TourPackage: p, Itinerary: items}, nil
}

func (r *TourRepo) Create(ctx context.Context, p tour.TourPackage) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO tour_packages
		  (destination_id, title, slug, description, price, price_label,
		   duration_days, max_participants, min_participants, package_type,
		   cover_image_url, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id`,
		p.DestinationID, p.Title, p.Slug, p.Description, p.Price, p.PriceLabel,
		p.DurationDays, p.MaxParticipants, p.MinParticipants,
		p.PackageType, p.CoverImageURL, p.IsActive,
	).Scan(&id)
	return id, err
}

func (r *TourRepo) Update(ctx context.Context, id string, p tour.TourPackage) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tour_packages SET
		  destination_id=$2, title=$3, slug=$4, description=$5, price=$6,
		  price_label=$7, duration_days=$8, max_participants=$9,
		  min_participants=$10, package_type=$11, cover_image_url=$12,
		  is_active=$13, updated_at=NOW()
		WHERE id=$1`,
		id, p.DestinationID, p.Title, p.Slug, p.Description, p.Price, p.PriceLabel,
		p.DurationDays, p.MaxParticipants, p.MinParticipants,
		p.PackageType, p.CoverImageURL, p.IsActive,
	)
	return err
}

func (r *TourRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tour_packages WHERE id=$1`, id)
	return err
}

func (r *TourRepo) ListDestinations(ctx context.Context) ([]tour.Destination, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, country, description, image_url FROM destinations ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var destinations []tour.Destination
	for rows.Next() {
		var d tour.Destination
		if err := rows.Scan(&d.ID, &d.Name, &d.Country, &d.Description, &d.ImageURL); err != nil {
			continue
		}
		destinations = append(destinations, d)
	}
	if destinations == nil {
		destinations = []tour.Destination{}
	}
	return destinations, nil
}

func (r *TourRepo) ListTestimonials(ctx context.Context, page, perPage int) ([]tour.Testimonial, error) {
	offset := (page - 1) * perPage
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, customer_name, content, rating, photo_url, created_at
		FROM testimonials
		WHERE is_published = true
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testimonials []tour.Testimonial
	for rows.Next() {
		var t tour.Testimonial
		if err := rows.Scan(&t.ID, &t.CustomerName, &t.Content, &t.Rating, &t.PhotoURL, &t.CreatedAt); err != nil {
			continue
		}
		testimonials = append(testimonials, t)
	}
	if testimonials == nil {
		testimonials = []tour.Testimonial{}
	}
	return testimonials, nil
}

func (r *TourRepo) ListGallery(ctx context.Context, packageID string) ([]tour.GalleryImage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, image_url, caption, sort_order
		FROM package_galleries
		WHERE tour_package_id = $1
		ORDER BY sort_order ASC, created_at ASC`, packageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gallery []tour.GalleryImage
	for rows.Next() {
		var g tour.GalleryImage
		if err := rows.Scan(&g.ID, &g.ImageURL, &g.Caption, &g.SortOrder); err != nil {
			continue
		}
		gallery = append(gallery, g)
	}
	if gallery == nil {
		gallery = []tour.GalleryImage{}
	}
	return gallery, nil
}

func (r *TourRepo) AddGalleryImage(ctx context.Context, packageID, imageURL string, caption *string, sortOrder int) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO package_galleries (tour_package_id, image_url, caption, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		packageID, imageURL, caption, sortOrder,
	).Scan(&id)
	return id, err
}

func (r *TourRepo) DeleteGalleryImage(ctx context.Context, packageID, imageID string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM package_galleries WHERE id=$1 AND tour_package_id=$2`, imageID, packageID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *TourRepo) ListItinerary(ctx context.Context, packageID string) ([]tour.ItineraryItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, day_number, title, COALESCE(description,''), COALESCE(location,''),
		       COALESCE(start_time,''), COALESCE(end_time,''), COALESCE(activity_type,''), sort_order
		FROM itinerary_items
		WHERE tour_package_id = $1
		ORDER BY day_number ASC, sort_order ASC`, packageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []tour.ItineraryItem
	for rows.Next() {
		var it tour.ItineraryItem
		if err := rows.Scan(&it.ID, &it.DayNumber, &it.Title, &it.Description,
			&it.Location, &it.StartTime, &it.EndTime, &it.ActivityType, &it.SortOrder); err != nil {
			continue
		}
		items = append(items, it)
	}
	if items == nil {
		items = []tour.ItineraryItem{}
	}
	return items, nil
}

func (r *TourRepo) AddItineraryItem(ctx context.Context, packageID string, item tour.ItineraryItem) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO itinerary_items
		  (tour_package_id, day_number, title, description, location, start_time, end_time, activity_type, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id`,
		packageID,
		item.DayNumber,
		item.Title,
		nullString(item.Description),
		nullString(item.Location),
		nullString(item.StartTime),
		nullString(item.EndTime),
		nullString(item.ActivityType),
		item.SortOrder,
	).Scan(&id)
	return id, err
}

func (r *TourRepo) UpdateItineraryItem(ctx context.Context, packageID, itemID string, item tour.ItineraryItem) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE itinerary_items SET
		  day_number=$3, title=$4, description=$5, location=$6,
		  start_time=$7, end_time=$8, activity_type=$9, sort_order=$10
		WHERE id=$1 AND tour_package_id=$2`,
		itemID, packageID,
		item.DayNumber,
		item.Title,
		nullString(item.Description),
		nullString(item.Location),
		nullString(item.StartTime),
		nullString(item.EndTime),
		nullString(item.ActivityType),
		item.SortOrder,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *TourRepo) DeleteItineraryItem(ctx context.Context, packageID, itemID string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM itinerary_items WHERE id=$1 AND tour_package_id=$2`, itemID, packageID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
