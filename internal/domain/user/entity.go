package user

import "time"

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
	ExperienceYears int       `json:"experience_years"`
	Specialization  string    `json:"specialization"`
	EmergencyPhone  string    `json:"emergency_phone"`
	CreatedAt       time.Time `json:"created_at"`
	// Joined from users
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}
