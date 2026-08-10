package portaluser

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// PortalUser is the central portal identity for a customer. One PortalUser can
// own many participants (one per tour) so a returning customer keeps the same
// login across trips (v2.0 F1).
type PortalUser struct {
	ID           string    `json:"id"`
	Phone        string    `json:"phone"`
	PasswordHash string    `json:"-"` // never serialize the hash
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// VerifyPassword reports whether pw matches the stored hash (§14.4).
//
// An account with no hash refuses every password. That case is not theoretical:
// a portal identity written outside the conversion flow would have an empty
// hash, and an empty-hash account that accepted an empty password would let
// anyone in with the phone number alone.
func (p *PortalUser) VerifyPassword(pw string) bool {
	if p == nil || p.PasswordHash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(p.PasswordHash), []byte(pw)) == nil
}
