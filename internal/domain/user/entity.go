package user

import (
	"context"
	"time"
)

// Roles lists the four staff roles of PRD §5.3, in privilege order. Keep in sync
// with the users_role_check constraint in db/migrations/003_prd_schema.sql — an
// account outside this set authenticates but is refused by every RBAC group.
var Roles = []string{"super_admin", "admin", "konsultan", "tour_leader"}

// AdminRoles are the roles that receive the admin notifications of §17.2.2.
//
// super_admin belongs here: it is the most privileged role, not a separate kind
// of account, and leaving it out meant whoever held it was blind to new leads,
// payment proofs and incoming documents. Three call sites each decided this for
// themselves and two of them decided it wrong, so it is decided once here.
var AdminRoles = []string{"admin", "super_admin"}

// ListAdmins returns every active user who should receive an admin
// notification. Errors are swallowed per role: a notification to the admins is
// best-effort, and one unreadable role should not silence the others.
func ListAdmins(ctx context.Context, repo Repository) []User {
	if repo == nil {
		return nil
	}
	var out []User
	for _, role := range AdminRoles {
		if users, err := repo.ListByRole(ctx, role); err == nil {
			out = append(out, users...)
		}
	}
	return out
}

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // jangan serialize password
	Role      string    `json:"role"`
	Phone     string    `json:"phone"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TourLeader struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Bio             string    `json:"bio"`
	PhotoPath       string    `json:"photo_path"`
	ExperienceYears int       `json:"experience_years" validate:"omitempty,gte=0"`
	Specialization  string    `json:"specialization"`
	EmergencyPhone  string    `json:"emergency_phone" validate:"omitempty,phone_id"`
	CreatedAt       time.Time `json:"created_at"`
	// Joined from users
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}
