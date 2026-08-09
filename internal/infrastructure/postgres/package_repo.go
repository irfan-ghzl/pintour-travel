package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	pkg "github.com/irfan-ghzl/pintour-travel/internal/domain/package"
)

type packageRepo struct{ db *sql.DB }

func NewPackageRepo(db *sql.DB) pkg.Repository { return &packageRepo{db} }

func (r *packageRepo) Create(ctx context.Context, p *pkg.Package) error {
	if len(p.Facilities) == 0 {
		p.Facilities = []byte("[]")
	}
	q := `INSERT INTO packages
		(id, name, slug, destination, category, duration_days, description,
		 itinerary, requirements, facilities, terms_conditions, visa_info,
		 base_price, is_active, thumbnail, created_by, created_at, updated_at)
		VALUES (gen_random_uuid(), $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW(),NOW())
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q,
		p.Name, p.Slug, p.Destination, p.Category, p.DurationDays, p.Description,
		p.Itinerary, p.Requirements, p.Facilities, p.TermsConditions, p.VisaInfo,
		p.BasePrice, p.IsActive, p.Thumbnail, p.CreatedBy,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *packageRepo) Update(ctx context.Context, p *pkg.Package) error {
	if len(p.Facilities) == 0 {
		p.Facilities = []byte("[]")
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE packages SET name=$1,slug=$2,destination=$3,category=$4,duration_days=$5,
		description=$6,itinerary=$7,requirements=$8,facilities=$9,
		terms_conditions=$10,visa_info=$11,base_price=$12,is_active=$13,
		thumbnail=$14,updated_at=NOW()
		WHERE id=$15 AND deleted_at IS NULL`,
		p.Name, p.Slug, p.Destination, p.Category, p.DurationDays,
		p.Description, p.Itinerary, p.Requirements, p.Facilities,
		p.TermsConditions, p.VisaInfo, p.BasePrice, p.IsActive,
		p.Thumbnail, p.ID,
	)
	return err
}

// Delete performs soft delete per §13.1 — jangan hapus fisik.
func (r *packageRepo) Delete(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE packages SET deleted_at=$1,updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL`, now, id)
	return err
}

func (r *packageRepo) GetByID(ctx context.Context, id string) (*pkg.Package, error) {
	return r.scanOne(ctx, `SELECT `+packageCols+` FROM packages WHERE id=$1 AND deleted_at IS NULL`, id)
}

func (r *packageRepo) GetBySlug(ctx context.Context, slug string) (*pkg.Package, error) {
	return r.scanOne(ctx, `SELECT `+packageCols+` FROM packages WHERE slug=$1 AND deleted_at IS NULL`, slug)
}

func (r *packageRepo) List(ctx context.Context, f pkg.Filter) ([]pkg.Package, int, error) {
	where := "WHERE deleted_at IS NULL"
	args := []interface{}{}
	i := 1

	if f.IsActive != nil {
		where += fmt.Sprintf(" AND is_active=$%d", i)
		args = append(args, *f.IsActive)
		i++
	}
	if f.Category != nil {
		where += fmt.Sprintf(" AND category=$%d", i)
		args = append(args, *f.Category)
		i++
	}
	if f.Destination != nil {
		where += fmt.Sprintf(" AND destination ILIKE $%d", i)
		args = append(args, "%"+*f.Destination+"%")
		i++
	}
	if f.PriceMin != nil {
		where += fmt.Sprintf(" AND base_price>=$%d", i)
		args = append(args, *f.PriceMin)
		i++
	}
	if f.PriceMax != nil {
		where += fmt.Sprintf(" AND base_price<=$%d", i)
		args = append(args, *f.PriceMax)
		i++
	}
	// FR-CAT-02: filter durasi perjalanan
	if f.DurationMin != nil {
		where += fmt.Sprintf(" AND duration_days>=$%d", i)
		args = append(args, *f.DurationMin)
		i++
	}
	if f.DurationMax != nil {
		where += fmt.Sprintf(" AND duration_days<=$%d", i)
		args = append(args, *f.DurationMax)
		i++
	}
	// FR-CAT-02: filter by departure month via JOIN package_batches
	if f.DepartureMonth != nil && *f.DepartureMonth != "" {
		where += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM package_batches pb2
			WHERE pb2.package_id=id
			AND TO_CHAR(pb2.departure_date,'YYYY-MM')=$%d
			AND pb2.status='tersedia'
			AND pb2.deleted_at IS NULL
		)`, i)
		args = append(args, *f.DepartureMonth)
		i++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM packages "+where, args...).Scan(&total); err != nil {
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
	q := fmt.Sprintf("SELECT "+packageCols+" FROM packages %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", where, i, i+1)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var packages []pkg.Package
	for rows.Next() {
		var p pkg.Package
		if err := scanPackage(rows, &p); err != nil {
			return nil, 0, err
		}
		packages = append(packages, p)
	}
	return packages, total, rows.Err()
}

func (r *packageRepo) scanOne(ctx context.Context, q string, args ...interface{}) (*pkg.Package, error) {
	row := r.db.QueryRowContext(ctx, q, args...)
	var p pkg.Package
	if err := scanPackageRow(row, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

const packageCols = `id,name,slug,destination,category,duration_days,
	COALESCE(description,''),itinerary,requirements,COALESCE(facilities,'[]'::jsonb),
	COALESCE(terms_conditions,''),COALESCE(visa_info,''),
	base_price,is_active,COALESCE(thumbnail,''),COALESCE(created_by::text,''),
	created_at,updated_at`

func scanPackage(rows *sql.Rows, p *pkg.Package) error {
	return rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Destination, &p.Category,
		&p.DurationDays, &p.Description, &p.Itinerary, &p.Requirements,
		&p.Facilities, &p.TermsConditions, &p.VisaInfo,
		&p.BasePrice, &p.IsActive, &p.Thumbnail, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
}

func scanPackageRow(row *sql.Row, p *pkg.Package) error {
	return row.Scan(&p.ID, &p.Name, &p.Slug, &p.Destination, &p.Category,
		&p.DurationDays, &p.Description, &p.Itinerary, &p.Requirements,
		&p.Facilities, &p.TermsConditions, &p.VisaInfo,
		&p.BasePrice, &p.IsActive, &p.Thumbnail, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
}

// ─── PackageImage ──────────────────────────────────────────────────────────────

type packageImageRepo struct{ db *sql.DB }

func NewPackageImageRepo(db *sql.DB) pkg.ImageRepository { return &packageImageRepo{db} }

func (r *packageImageRepo) Add(ctx context.Context, img *pkg.PackageImage) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO package_images(id,package_id,file_path,alt_text,sort_order,is_thumbnail,created_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,NOW())
		RETURNING id,created_at`,
		img.PackageID, img.FilePath, img.AltText, img.SortOrder, img.IsThumbnail,
	).Scan(&img.ID, &img.CreatedAt)
}

// Delete performs soft delete on package image.
func (r *packageImageRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE package_images SET deleted_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *packageImageRepo) ListByPackage(ctx context.Context, packageID string) ([]pkg.PackageImage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id,package_id,file_path,COALESCE(alt_text,''),sort_order,is_thumbnail,created_at
		FROM package_images
		WHERE package_id=$1 AND deleted_at IS NULL
		ORDER BY sort_order`, packageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var imgs []pkg.PackageImage
	for rows.Next() {
		var img pkg.PackageImage
		if err := rows.Scan(&img.ID, &img.PackageID, &img.FilePath, &img.AltText,
			&img.SortOrder, &img.IsThumbnail, &img.CreatedAt); err != nil {
			return nil, err
		}
		imgs = append(imgs, img)
	}
	return imgs, rows.Err()
}

// ─── PackageBatch ─────────────────────────────────────────────────────────────

type packageBatchRepo struct{ db *sql.DB }

func NewPackageBatchRepo(db *sql.DB) pkg.BatchRepository { return &packageBatchRepo{db} }

func (r *packageBatchRepo) Create(ctx context.Context, b *pkg.PackageBatch) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO package_batches
		(id,package_id,departure_date,return_date,quota,price_single,price_double,
		 price_triple,status,tour_leader_id,wa_group_link,created_at,updated_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),NOW())
		RETURNING id,created_at,updated_at`,
		b.PackageID, b.DepartureDate, b.ReturnDate, b.Quota,
		b.PriceSingle, b.PriceDouble, b.PriceTriple, b.Status, b.TourLeaderID, b.WaGroupLink,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
}

func (r *packageBatchRepo) Update(ctx context.Context, b *pkg.PackageBatch) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE package_batches SET departure_date=$1,return_date=$2,quota=$3,
		price_single=$4,price_double=$5,price_triple=$6,status=$7,
		tour_leader_id=$8,wa_group_link=$9,updated_at=NOW()
		WHERE id=$10 AND deleted_at IS NULL`,
		b.DepartureDate, b.ReturnDate, b.Quota, b.PriceSingle, b.PriceDouble,
		b.PriceTriple, b.Status, b.TourLeaderID, b.WaGroupLink, b.ID)
	return err
}

const batchCols = `pb.id,pb.package_id,pb.departure_date,pb.return_date,pb.quota,
	pb.price_single,pb.price_double,pb.price_triple,pb.status,pb.tour_leader_id,
	pb.created_at,pb.updated_at,p.name,COALESCE(u.name,''),COALESCE(pb.wa_group_link,'')`

func scanBatch(s interface{ Scan(...interface{}) error }, b *pkg.PackageBatch) error {
	return s.Scan(&b.ID, &b.PackageID, &b.DepartureDate, &b.ReturnDate,
		&b.Quota, &b.PriceSingle, &b.PriceDouble, &b.PriceTriple, &b.Status,
		&b.TourLeaderID, &b.CreatedAt, &b.UpdatedAt, &b.PackageName, &b.TourLeaderName,
		&b.WaGroupLink)
}

func (r *packageBatchRepo) GetByID(ctx context.Context, id string) (*pkg.PackageBatch, error) {
	var b pkg.PackageBatch
	row := r.db.QueryRowContext(ctx, `
		SELECT `+batchCols+`
		FROM package_batches pb
		JOIN packages p ON p.id=pb.package_id
		LEFT JOIN users u ON u.id=pb.tour_leader_id
		WHERE pb.id=$1 AND pb.deleted_at IS NULL`, id)
	if err := scanBatch(row, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *packageBatchRepo) List(ctx context.Context, f pkg.BatchFilter) ([]pkg.PackageBatch, error) {
	where := "WHERE pb.deleted_at IS NULL"
	args := []interface{}{}
	i := 1
	if f.PackageID != nil {
		where += fmt.Sprintf(" AND pb.package_id=$%d", i)
		args = append(args, *f.PackageID)
		i++
	}
	if f.Status != nil {
		where += fmt.Sprintf(" AND pb.status=$%d", i)
		args = append(args, *f.Status)
		i++
	}
	_ = i
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+batchCols+`
		FROM package_batches pb
		JOIN packages p ON p.id=pb.package_id
		LEFT JOIN users u ON u.id=pb.tour_leader_id
		`+where+` ORDER BY pb.departure_date`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBatches(rows)
}

func (r *packageBatchRepo) ListByPackage(ctx context.Context, packageID string) ([]pkg.PackageBatch, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+batchCols+`
		FROM package_batches pb
		JOIN packages p ON p.id=pb.package_id
		LEFT JOIN users u ON u.id=pb.tour_leader_id
		WHERE pb.package_id=$1 AND pb.deleted_at IS NULL
		ORDER BY pb.departure_date`, packageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBatches(rows)
}

func scanBatches(rows *sql.Rows) ([]pkg.PackageBatch, error) {
	var batches []pkg.PackageBatch
	for rows.Next() {
		var b pkg.PackageBatch
		if err := scanBatch(rows, &b); err != nil {
			return nil, err
		}
		batches = append(batches, b)
	}
	return batches, rows.Err()
}
