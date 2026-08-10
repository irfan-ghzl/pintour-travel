package httpdelivery

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/irfan-ghzl/pintour-travel/internal/auth"
	"github.com/irfan-ghzl/pintour-travel/internal/safe"
	"github.com/labstack/echo/v4"
)

// JWTCookieName — nama cookie httpOnly untuk JWT admin/staff (§19.1).
const JWTCookieName = "pintour_session"

// Request body ceilings.
const (
	// maxJSONBody caps a request whose payload is a handful of fields. No login,
	// lead form, or webhook notification comes close to a megabyte, so anything
	// past it is a mistake or an attempt to exhaust memory.
	maxJSONBody = 1 << 20 // 1 MB
	// maxUploadBody caps a multipart request — the only kind that legitimately
	// carries a file: a participant document or payment proof (§16.2, 5 MB) plus
	// its multipart framing. A JSON endpoint sent multipart gets this larger
	// ceiling as well; it is still a ceiling, and the decode then fails on the
	// content instead of on the size.
	maxUploadBody = 12 << 20 // 12 MB
)

// recoverPanics turns a panic anywhere in the chain into a 500 carrying the
// API's error envelope, and reports it with the stack that raised it.
//
// Without this the HTTP server drops the connection mid-response and the caller
// sees a transport failure with nothing in the application log to explain it.
func recoverPanics() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				safe.LogPanic(c.Request().Method+" "+c.Request().URL.Path, v)
				if c.Response().Committed {
					// A partial response is already on the wire; there is nothing
					// left to say to this client that would not corrupt it.
					return
				}
				err = c.JSON(http.StatusInternalServerError, serverErrEnvelope())
			}()
			return next(c)
		}
	}
}

// bodyLimit refuses a request whose body is larger than the ceiling for its
// kind. One middleware covers every route rather than a limit per route, so an
// endpoint added later is capped by default instead of by remembering to.
//
// A request that declares its size is refused on the declaration alone, before
// a byte is read. One that does not — chunked transfer encoding — is read as
// far as the ceiling and no further: for JSON that peek costs a megabyte and
// keeps the answer a 413 rather than an unexplained decode failure inside a
// handler; for a file it would cost twelve, so that case streams under a cap
// instead. Either way the request is bounded before a handler sees it.
func bodyLimit() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			limit := int64(maxJSONBody)
			if strings.HasPrefix(strings.ToLower(req.Header.Get(echo.HeaderContentType)), "multipart/") {
				limit = maxUploadBody
			}
			if req.ContentLength > limit {
				return requestTooLarge(c, limit)
			}
			if req.ContentLength < 0 && req.Body != nil {
				if limit == maxUploadBody {
					// A file is too expensive to hold just to measure it, so it
					// is capped as it streams. The handler then fails on the
					// truncated upload rather than on its size — a worse answer
					// than 413, but bought with a megabyte instead of twelve.
					req.Body = http.MaxBytesReader(c.Response(), req.Body, limit)
					return next(c)
				}
				buffered, err := io.ReadAll(io.LimitReader(req.Body, limit+1))
				if err != nil {
					return badRequest(c, "gagal membaca isi permintaan")
				}
				if int64(len(buffered)) > limit {
					return requestTooLarge(c, limit)
				}
				req.Body = io.NopCloser(bytes.NewReader(buffered))
			}
			return next(c)
		}
	}
}

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

			// Portal tokens are signed with the same secret but carry only
			// participant_id (no user_id/role). Reject them here so a participant
			// token can never reach an admin/staff route (§19.1 privilege separation).
			if claims.UserID == "" || claims.Role == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "token bukan untuk akses admin/staff")
			}

			c.Set("claims", claims)
			c.Set("user_id", claims.UserID)
			c.Set("user_role", claims.Role)
			return next(c)
		}
	}
}

// RequireRole restricts a route (or route group) to the given staff roles.
// Must run AFTER JWTMiddleware, which populates "claims". A token whose role is
// not in the allow-list gets 403 — this is the server-side half of the RBAC;
// the sidebar only hides links, so without this an out-of-scope role could still
// reach the data by calling the API directly.
func RequireRole(roles ...string) echo.MiddlewareFunc {
	// The allow-list is built once per route group rather than walked per
	// request. domainUser.User.HasRole (§14.4) states the same rule on the
	// entity, but there is no account here to ask — only the role the token
	// carries, and constructing a User around it to scan a list would be a
	// slower way of saying the same thing.
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, _ := c.Get("user_role").(string)
			if role == "" || !allowed[role] {
				return c.JSON(http.StatusForbidden, errResponse("FORBIDDEN", "Akses tidak diizinkan untuk peran Anda"))
			}
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
