package postgres

import (
	"context"
	"database/sql"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/portaluser"
)

type portalUserRepo struct{ db dbtx }

func NewPortalUserRepo(db *sql.DB) portaluser.Repository { return &portalUserRepo{db} }

func (r *portalUserRepo) Create(ctx context.Context, u *portaluser.PortalUser) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO portal_users (id, phone, password_hash, name, email, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at`,
		u.Phone, u.PasswordHash, u.Name, u.Email,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *portalUserRepo) GetByID(ctx context.Context, id string) (*portaluser.PortalUser, error) {
	var u portaluser.PortalUser
	err := r.db.QueryRowContext(ctx, `
		SELECT id, phone, password_hash, name, COALESCE(email,''), created_at, updated_at
		FROM portal_users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Phone, &u.PasswordHash, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *portalUserRepo) GetByPhone(ctx context.Context, phone string) (*portaluser.PortalUser, error) {
	var u portaluser.PortalUser
	err := r.db.QueryRowContext(ctx, `
		SELECT id, phone, password_hash, name, COALESCE(email,''), created_at, updated_at
		FROM portal_users WHERE phone=$1`, phone,
	).Scan(&u.ID, &u.Phone, &u.PasswordHash, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
