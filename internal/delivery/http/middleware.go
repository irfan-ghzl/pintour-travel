package httpdelivery

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/irfan-ghzl/pintour-travel/internal/auth"
	"github.com/labstack/echo/v4"
)

// JWTMiddleware validates Bearer tokens and attaches claims to the Echo context.
func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
			}

			claims := &auth.Claims{}
			token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(http.StatusUnauthorized, "unexpected signing method")
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set("claims", claims)
			c.Set("user_id", claims.UserID)
			c.Set("user_role", claims.Role)
			return next(c)
		}
	}
}
