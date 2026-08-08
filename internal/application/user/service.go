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

// dummyBcryptHash is a valid bcrypt hash (cost 12) of an arbitrary string,
// used to equalize login timing when the supplied email does not exist.
const dummyBcryptHash = "$2b$12$ev6WdpzmJ5M1FbM4AASRRuz3ih/M0KYrT0aoG4wFqM2gRj.rWbo8q"

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
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
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
		// Perform a dummy bcrypt comparison so the response time is the same
		// whether or not the email exists, preventing user enumeration via timing.
		_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(req.Password))
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
