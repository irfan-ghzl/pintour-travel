package user

import "context"

// Repository is the persistence abstraction for the user domain.
type Repository interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
}
