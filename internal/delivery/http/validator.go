package httpdelivery

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// CustomValidator wraps go-playground/validator (PRD §18 + §19.3 input validation).
type CustomValidator struct {
	v *validator.Validate
}

// NewValidator creates a Validator backed by go-playground/validator.
func NewValidator() *CustomValidator {
	return &CustomValidator{v: validator.New()}
}

// Validate is invoked by Echo when c.Validate() is called.
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.v.Struct(i)
}

// RegisterValidator wires the validator into the Echo instance.
func RegisterValidator(e *echo.Echo) {
	e.Validator = NewValidator()
}
