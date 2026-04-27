package user

// User is the identity aggregate.
type User struct {
	ID       string
	Name     string
	Email    string
	Password string
	Role     string
	IsActive bool
}
