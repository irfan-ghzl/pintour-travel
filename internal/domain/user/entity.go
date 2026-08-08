package user

import "time"

// Roles lists the four staff roles of PRD §5.3, in privilege order. Keep in sync
// with the users_role_check constraint in db/migrations/003_prd_schema.sql — an
// account outside this set authenticates but is refused by every RBAC group.
var Roles = []string{"super_admin", "admin", "konsultan", "tour_leader"}

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
