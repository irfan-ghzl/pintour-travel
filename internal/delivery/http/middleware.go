package httpdelivery

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/irfan-ghzl/pintour-travel/internal/auth"
	"github.com/labstack/echo/v4"
)

// JWTCookieName — nama cookie httpOnly untuk JWT admin/staff (§19.1).
const JWTCookieName = "pintour_session"

// JWTMiddleware validates Bearer tokens or httpOnly session cookie (§19.1).
// Preferensi: cookie httpOnly (mencegah XSS) → fallback Authorization header.
func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenStr := ""

			// 1. Coba dari cookie httpOnly (preferensi PRD §19.1)
			if cookie, err := c.Cookie(JWTCookieName); err == nil && cookie.Value != "" {
				tokenStr = cookie.Value
			}

			// 2. Fallback ke Authorization header (untuk klien non-browser / Swagger)
			if tokenStr == "" {
				authHeader := c.Request().Header.Get("Authorization")
				if authHeader != "" {
					parts := strings.SplitN(authHeader, " ", 2)
					if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
						tokenStr = parts[1]
					}
				}
			}

			if tokenStr == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization (cookie atau Bearer token)")
			}

			claims := &auth.Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
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

// SetSessionCookie sets the httpOnly JWT cookie (§19.1).
// Secure flag mengikuti env: production = HTTPS only.
func SetSessionCookie(c echo.Context, token string, maxAgeSec int, prod bool) {
	c.SetCookie(&http.Cookie{
		Name:     JWTCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAgeSec,
		HttpOnly: true,
		Secure:   prod,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie removes the JWT cookie on logout.
func ClearSessionCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     JWTCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
