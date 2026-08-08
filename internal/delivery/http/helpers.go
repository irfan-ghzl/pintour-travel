package httpdelivery

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/irfan-ghzl/pintour-travel/internal/auth"
	"github.com/labstack/echo/v4"
)

// ─── Response wrappers ────────────────────────────────────────────────────────

func ok(data interface{}) map[string]interface{} {
	return map[string]interface{}{"success": true, "data": data}
}

func pageResponse(data interface{}, total, page, perPage int) map[string]interface{} {
	totalPages := (total + perPage - 1) / perPage
	return map[string]interface{}{
		"success": true,
		"data":    data,
		"meta": map[string]int{
			"page":        page,
			"limit":       perPage,
			"total":       total,
			"total_pages": totalPages,
		},
	}
}

func errResponse(code, message string) map[string]interface{} {
	return map[string]interface{}{"success": false, "error": code, "message": message}
}

func badRequest(c echo.Context, message string) error {
	return c.JSON(http.StatusBadRequest, errResponse("BAD_REQUEST", message))
}

func notFound(c echo.Context, message string) error {
	return c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", message))
}

// serverErrEnvelope is the body behind every 500: deliberately generic, so
// internal details (queries, paths, panic values) are not disclosed to the
// client. The panic-recovery middleware answers with it too, which is why it is
// separate from serverErr — that one also logs, and a recovered panic is
// already logged with its stack.
func serverErrEnvelope() map[string]interface{} {
	return errResponse("SERVER_ERROR", "Terjadi kesalahan internal")
}

func serverErr(c echo.Context, err error) error {
	c.Logger().Error(err)
	return c.JSON(http.StatusInternalServerError, serverErrEnvelope())
}

// requestTooLarge refuses a request whose body exceeds limit bytes (§ketahanan
// runtime).
func requestTooLarge(c echo.Context, limit int64) error {
	return c.JSON(http.StatusRequestEntityTooLarge, errResponse("REQUEST_TOO_LARGE",
		fmt.Sprintf("Ukuran permintaan melebihi batas %d KB", limit/1024)))
}

func forbidden(c echo.Context) error {
	return c.JSON(http.StatusForbidden, errResponse("FORBIDDEN", "Akses tidak diizinkan"))
}

// ─── JWT claim helpers ────────────────────────────────────────────────────────

func claimUserID(c echo.Context) string {
	if claims, ok := c.Get("claims").(*auth.Claims); ok {
		return claims.UserID
	}
	return ""
}

func claimRole(c echo.Context) string {
	if claims, ok := c.Get("claims").(*auth.Claims); ok {
		return claims.Role
	}
	return ""
}

// ─── Query helpers ────────────────────────────────────────────────────────────

func queryInt(c echo.Context, name string, defaultVal int) int {
	s := c.QueryParam(name)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}

// maxPerPage caps how many rows one request may ask a list endpoint for
// (§19.3). Without a ceiling a client can ask for a whole table in one query and
// make the server materialise it.
const maxPerPage = 100

// queryPageSize reads a page-size parameter, falling back to defaultVal when it
// is absent or unreadable, and clamping anything above maxPerPage down to it. It
// clamps rather than refuses: an over-large page is a caller asking for too much
// of a resource they are entitled to, not a malformed request, and the response
// reports the size actually served in its meta.
func queryPageSize(c echo.Context, name string, defaultVal int) int {
	if v := queryInt(c, name, defaultVal); v <= maxPerPage {
		return v
	}
	return maxPerPage
}

func queryStringPtr(c echo.Context, name string) *string {
	v := c.QueryParam(name)
	if v == "" {
		return nil
	}
	return &v
}

func queryFloat64Ptr(c echo.Context, name string) *float64 {
	s := c.QueryParam(name)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

func queryIntPtr(c echo.Context, name string) *int {
	s := c.QueryParam(name)
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return nil
	}
	return &v
}
