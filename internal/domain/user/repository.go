package user

import "context"

type Repository interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	Create(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	// UpdatePassword menulis kata sandi, dan hanya itu.
	//
	// Update sengaja tidak menyentuh kolom password: ia melayani penyuntingan
	// profil, dan sebuah profil yang tersimpan tidak boleh berisiko menimpa
	// kredensial. Akibatnya kedua jalur reset — oleh super admin dan lewat tautan
	// surel — memanggil Update setelah menetapkan hash baru, lalu melaporkan
	// berhasil tanpa satu pun kolom berubah.
	UpdatePassword(ctx context.Context, id, hashedPassword string) error
	Deactivate(ctx context.Context, id string) error
	ListByRole(ctx context.Context, role string) ([]User, error)
	ListKonsultan(ctx context.Context) ([]User, error)
}

type TourLeaderRepository interface {
	Create(ctx context.Context, tl *TourLeader) error
	GetByUserID(ctx context.Context, userID string) (*TourLeader, error)
	Update(ctx context.Context, tl *TourLeader) error
	List(ctx context.Context) ([]TourLeader, error)
}
