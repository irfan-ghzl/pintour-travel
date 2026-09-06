package postgres

import (
	"context"
	"database/sql"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/user"
)

type userRepo struct{ db *sql.DB }

func NewUserRepo(db *sql.DB) user.Repository { return &userRepo{db} }

const userCols = `id,name,email,password,role,COALESCE(phone,''),is_active,created_at,updated_at`

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := r.db.QueryRowContext(ctx,
		`SELECT `+userCols+` FROM users WHERE email=$1 AND is_active=true`, email,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &u.Phone, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*user.User, error) {
	var u user.User
	err := r.db.QueryRowContext(ctx,
		`SELECT `+userCols+` FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &u.Phone, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) Create(ctx context.Context, u *user.User) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO users(id,name,email,password,role,phone,is_active,created_at,updated_at)
		VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,NOW(),NOW())
		RETURNING id,created_at,updated_at`,
		u.Name, u.Email, u.Password, u.Role, u.Phone, u.IsActive,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *userRepo) Update(ctx context.Context, u *user.User) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET name=$1,email=$2,role=$3,phone=$4,updated_at=NOW()
		WHERE id=$5`,
		u.Name, u.Email, u.Role, u.Phone, u.ID)
	return err
}

func (r *userRepo) UpdatePassword(ctx context.Context, id, hashedPassword string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET password=$1,updated_at=NOW() WHERE id=$2`, hashedPassword, id)
	if err != nil {
		return err
	}
	// Baris yang tidak tersentuh dilaporkan sebagai galat, bukan sebagai sukses
	// yang sunyi. Cacat sebelumnya justru berbentuk begitu: pemanggil menerima
	// kabar berhasil sementara kata sandi lama masih berlaku.
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *userRepo) Deactivate(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET is_active=false,updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *userRepo) ListByRole(ctx context.Context, role string) ([]user.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+userCols+` FROM users WHERE role=$1 AND is_active=true ORDER BY name`, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (r *userRepo) ListKonsultan(ctx context.Context) ([]user.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+userCols+` FROM users WHERE role='konsultan' AND is_active=true ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

func scanUsers(rows *sql.Rows) ([]user.User, error) {
	var list []user.User
	for rows.Next() {
		var u user.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Role,
			&u.Phone, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}
