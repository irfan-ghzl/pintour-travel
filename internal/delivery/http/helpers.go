package httpdelivery

import (
	"strconv"

	"github.com/labstack/echo/v4"
)

// queryInt reads an integer query parameter with a fallback default value.
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

// queryStringPtr returns nil if the query param is empty, otherwise a pointer to its value.
func queryStringPtr(c echo.Context, name string) *string {
	v := c.QueryParam(name)
	if v == "" {
		return nil
	}
	return &v
}

// queryFloat64Ptr parses a float64 query param; returns nil if absent or invalid.
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

// queryIntPtr parses an integer query param; returns nil if absent or invalid.
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
