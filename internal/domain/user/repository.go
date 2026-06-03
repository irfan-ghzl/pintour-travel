package user

import "context"

type Repository interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	Create(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	Deactivate(ctx context.Context, id string) error
	ListByRole(ctx context.Context, role string) ([]User, error)
	ListKonsultan(ctx context.Context) ([]User, error)
	CountActiveleadsByConsultant(ctx context.Context, consultantID string) (int, error)
}

type TourLeaderRepository interface {
	Create(ctx context.Context, tl *TourLeader) error
	GetByUserID(ctx context.Context, userID string) (*TourLeader, error)
	Update(ctx context.Context, tl *TourLeader) error
	List(ctx context.Context) ([]TourLeader, error)
}
