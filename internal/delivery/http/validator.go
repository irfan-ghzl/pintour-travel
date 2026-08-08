package httpdelivery

import (
	"github.com/labstack/echo/v4"
)

// CustomValidator adapts the shared validator to Echo's Validator interface,
// which PRD §18 asks for. No handler calls c.Validate() today — bindJSON is the
// live validation path (see binding.go) — so this exists to guarantee that one
// which does gets exactly the same rules, custom tags included, rather than a
// second unconfigured validator or a nil-Validator panic.
type CustomValidator struct{}

// NewValidator creates a Validator backed by go-playground/validator.
func NewValidator() *CustomValidator {
	return &CustomValidator{}
}

// Validate is invoked by Echo when c.Validate() is called.
func (cv *CustomValidator) Validate(i interface{}) error {
	return validatePayload(i)
}

// RegisterValidator wires the validator into the Echo instance.
func RegisterValidator(e *echo.Echo) {
	e.Validator = NewValidator()
}
