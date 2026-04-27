package usersvc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/irfan-ghzl/pintour-travel/internal/auth"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles authentication use cases.
type UserService struct {
	repo      user.Repository
	jwtSecret string
	jwtExpiry time.Duration
}

// NewUserService creates a new UserService.
func NewUserService(repo user.Repository, jwtSecret string, jwtExpiryHours int) *UserService {
	return &UserService{
		repo:      repo,
		jwtSecret: jwtSecret,
		jwtExpiry: time.Duration(jwtExpiryHours) * time.Hour,
	}
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

// Login validates credentials and returns a signed JWT on success.
func (s *UserService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, int, error) {
	u, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("login query: %w", err)
	}
	if u == nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("invalid credentials")
	}

	expiresAt := time.Now().Add(s.jwtExpiry)
	claims := auth.Claims{
		UserID: u.ID,
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("generate token: %w", err)
	}

	return &LoginResponse{
		Token:     signed,
		UserID:    u.ID,
		Name:      u.Name,
		Role:      u.Role,
		ExpiresAt: expiresAt,
	}, http.StatusOK, nil
}
