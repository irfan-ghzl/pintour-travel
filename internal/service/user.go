package service

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/irfan-ghzl/pintour-travel/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

// UserRow represents a simplified user record (mirrors what sqlc would generate).
type UserRow struct {
	ID        uuid.UUID
	Name      string
	Email     string
	Password  string
	Role      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserQuerier is the subset of the sqlc Querier interface used by UserService.
type UserQuerier interface {
	GetUserByEmail(ctx context.Context, email string) (UserRow, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (UserRow, error)
	CreateUser(ctx context.Context, arg interface{}) (UserRow, error)
}

// UserService handles authentication and user management.
type UserService struct {
	db        *sql.DB
	jwtSecret string
	jwtExpiry time.Duration
}

// NewUserService creates a new UserService.
func NewUserService(db *sql.DB, jwtSecret string, jwtExpiryHours int) *UserService {
	return &UserService{
		db:        db,
		jwtSecret: jwtSecret,
		jwtExpiry: time.Duration(jwtExpiryHours) * time.Hour,
	}
}

// HashPassword returns the bcrypt hash of the plain-text password.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(b), nil
}

// CheckPassword compares a plain-text password against its bcrypt hash.
func CheckPassword(plain, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

// GenerateToken creates a signed JWT for the given user.
func (s *UserService) GenerateToken(userID, role string) (string, error) {
	claims := middleware.Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return signed, nil
}

// LoginRequest holds credentials for login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is returned on successful login.
type LoginResponse struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Login validates credentials against the database and returns a JWT on success.
func (s *UserService) Login(ctx context.Context, db *sql.DB, req LoginRequest) (*LoginResponse, int, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, email, password, role, is_active FROM users WHERE email = $1 AND is_active = true`,
		req.Email,
	)

	var u UserRow
	var isActive bool
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &isActive); err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusUnauthorized, fmt.Errorf("invalid credentials")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("login query: %w", err)
	}

	if err := CheckPassword(req.Password, u.Password); err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("invalid credentials")
	}

	expiresAt := time.Now().Add(s.jwtExpiry)
	token, err := s.GenerateToken(u.ID.String(), u.Role)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return &LoginResponse{
		Token:     token,
		UserID:    u.ID.String(),
		Name:      u.Name,
		Role:      u.Role,
		ExpiresAt: expiresAt,
	}, http.StatusOK, nil
}
