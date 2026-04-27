package postgres

import (
	"context"
	"database/sql"

	"github.com/irfan-ghzl/pintour-travel/internal/domain/user"
)

// UserRepo implements user.Repository against PostgreSQL.
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, email, password, role, is_active FROM users WHERE email = $1 AND is_active = true`,
		email,
	)
	var u user.User
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &u.IsActive); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}
