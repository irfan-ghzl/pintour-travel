package httpdelivery

// Request binding and input validation (PRD §19.3).
//
// Every handler decodes its JSON body through bindJSON, and bindJSON runs the
// go-playground/validator over the result. That is what makes §19.3 — "semua
// input dari user divalidasi menggunakan go-playground/validator di layer
// handler" — true for the whole API rather than for whichever handlers
// remembered to ask: attaching a `validate` tag to a payload field is all it
// takes to have bad values refused before they reach a repository.
//
// Failures come back as validationFailure, which invalidPayload turns into a 400
// naming the offending fields. Field names in those messages are the JSON names
// the client sent, not the Go field names, so the message points at something
// the caller can actually find in their payload.

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/calendar"
	"github.com/irfan-ghzl/pintour-travel/internal/domain/document"
	domainLead "github.com/irfan-ghzl/pintour-travel/internal/domain/lead"
	domainParticipant "github.com/irfan-ghzl/pintour-travel/internal/domain/participant"
	domainUser "github.com/irfan-ghzl/pintour-travel/internal/domain/user"
	"github.com/labstack/echo/v4"
)

// payloadValidator is the process-wide validator. One instance, because it
// caches struct metadata per type and registering the custom rules twice would
// waste that cache.
var payloadValidator = newPayloadValidator()

func newPayloadValidator() *validator.Validate {
	v := validator.New()

	// Report the JSON name of a field, so "phone harus diisi" matches the key the
	// client actually sent instead of naming the Go field it landed in.
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})

	// phone_id: an Indonesian mobile number in any of the shapes the public form,
	// the portal, and the CRM actually submit.
	_ = v.RegisterValidation("phone_id", func(fl validator.FieldLevel) bool {
		return isIndonesianPhone(fl.Field().String())
	})

	// calendar.Date carries its day in an unexported field, which the validator
	// cannot see — so `required` on a date read it as present no matter what,
	// and an empty date picker submitted a batch with no departure. Handing over
	// the underlying instant makes `required` mean what it means everywhere else.
	v.RegisterCustomTypeFunc(func(field reflect.Value) any {
		if d, ok := field.Interface().(calendar.Date); ok {
			return d.Time()
		}
		return nil
	}, calendar.Date{})

	// One tag per vocabulary that already has a canonical list in the domain, so
	// the list lives in exactly one place instead of being restated in every
	// oneof= tag that needs it.
	registerVocabulary(v, "lead_status", domainLead.Statuses)
	registerVocabulary(v, "staff_role", domainUser.Roles)
	registerVocabulary(v, "document_type", document.Types)
	registerVocabulary(v, "room_type", domainParticipant.RoomTypes)

	return v
}

// registerVocabulary registers name as a tag accepting exactly the values in
// allowed. describeFieldError renders a failure the same way it renders oneof,
// which is what these tags are — with the list read from the domain rather than
// retyped in the struct tag.
func registerVocabulary(v *validator.Validate, name string, allowed []string) {
	set := make(map[string]struct{}, len(allowed))
	for _, value := range allowed {
		set[value] = struct{}{}
	}
	_ = v.RegisterValidation(name, func(fl validator.FieldLevel) bool {
		_, found := set[fl.Field().String()]
		return found
	})
	vocabularies[name] = strings.Join(allowed, ", ")
}

// vocabularies remembers each registered tag's allowed values, so an error
// message can list them the way an oneof failure does.
var vocabularies = map[string]string{}

// phoneSeparators are the characters people type inside a phone number. They
// carry no information, so they are dropped before the number is judged —
// without this, "0812-3456-7890" would be refused although every form in the
// app accepted it before validation existed.
var phoneSeparators = strings.NewReplacer(" ", "", "-", "", ".", "", "(", "", ")", "")

// isIndonesianPhone reports whether s is a WhatsApp-reachable Indonesian number
// in any of the forms the app accepts: 08…, 62…, or +62…, with or without
// separators. Note that it judges a cleaned copy — normalizePhone, which the
// handlers apply to the value they store, does not itself strip separators.
func isIndonesianPhone(s string) bool {
	phone := normalizePhone(phoneSeparators.Replace(strings.TrimSpace(s)))
	if !strings.HasPrefix(phone, "62") {
		return false
	}
	// 62 + 8-13 digits: the lower bound is the one CreateLead applied before this
	// tag existed; the upper is E.164's ceiling.
	if len(phone) < 10 || len(phone) > 15 {
		return false
	}
	for _, r := range phone {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// bindJSON decodes the request body into v and validates it against v's
// `validate` tags. A decode failure is returned as-is; a tag failure comes back
// as validationFailure. Pass the error to invalidPayload to answer the client.
func bindJSON(c echo.Context, v interface{}) error {
	if err := json.NewDecoder(c.Request().Body).Decode(v); err != nil {
		return err
	}
	return validatePayload(v)
}

// validatePayload applies the validate tags on v. Non-structs (a map, a slice)
// carry no tags and pass unexamined.
func validatePayload(v interface{}) error {
	err := payloadValidator.Struct(v)
	if err == nil {
		return nil
	}
	var notAStruct *validator.InvalidValidationError
	if errors.As(err, &notAStruct) {
		return nil
	}
	var fieldErrs validator.ValidationErrors
	if !errors.As(err, &fieldErrs) {
		return err
	}
	messages := make([]string, 0, len(fieldErrs))
	for _, fe := range fieldErrs {
		messages = append(messages, describeFieldError(fe))
	}
	return validationFailure{message: strings.Join(messages, "; ")}
}

// validationFailure is a payload that decoded cleanly but broke a validate tag.
// Its message names every field that failed and why.
type validationFailure struct{ message string }

func (e validationFailure) Error() string { return e.message }

// invalidPayload answers the 400 for a failed bindJSON. A broken validate tag
// names the field that broke it; anything else — a body that was not JSON, or a
// caller's own check failing with err still nil — falls back to fallback, which
// is the message that handler used before validation existed.
func invalidPayload(c echo.Context, err error, fallback string) error {
	var failure validationFailure
	if errors.As(err, &failure) {
		return badRequest(c, failure.Error())
	}
	// A date the API could not read says so, and quotes the value. The admin
	// forms bind a date picker straight into the body, so a date the decoder
	// refuses is the single most likely reason one of them fails — answering it
	// with the handler's generic fallback left the admin with a form that never
	// worked and no clue why.
	var badDate calendar.ParseError
	if errors.As(err, &badDate) {
		return badRequest(c, badDate.Error())
	}
	return badRequest(c, fallback)
}

// describeFieldError renders one failed tag as a sentence the client can act on.
// Unlisted tags fall back to naming the field, which is still more use than the
// library's default English sentence about namespaces.
func describeFieldError(fe validator.FieldError) string {
	field := fe.Field()
	if allowed, isVocabulary := vocabularies[fe.Tag()]; isVocabulary {
		return fmt.Sprintf("%s harus salah satu dari: %s", field, allowed)
	}
	switch fe.Tag() {
	case "required", "required_if", "required_with", "required_without":
		return field + " harus diisi"
	case "oneof":
		return fmt.Sprintf("%s harus salah satu dari: %s", field,
			strings.Join(strings.Fields(fe.Param()), ", "))
	case "email":
		return field + " bukan alamat email yang valid"
	case "phone_id":
		return field + " bukan nomor WhatsApp Indonesia yang valid (08…, 62…, atau +62…)"
	case "url":
		return field + " bukan URL yang valid"
	case "numeric":
		return field + " hanya boleh berisi angka"
	case "len":
		return fmt.Sprintf("%s harus tepat %s karakter", field, fe.Param())
	case "min":
		return fmt.Sprintf("%s minimal %s", field, quantity(fe))
	case "max":
		return fmt.Sprintf("%s maksimal %s", field, quantity(fe))
	case "gt":
		return fmt.Sprintf("%s harus lebih besar dari %s", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s minimal %s", field, fe.Param())
	case "lt":
		return fmt.Sprintf("%s harus lebih kecil dari %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s maksimal %s", field, fe.Param())
	default:
		return field + " tidak valid"
	}
}

// quantity spells out what a min/max bound counts, which depends on the field's
// kind: characters for text, items for a list, and a bare number otherwise.
func quantity(fe validator.FieldError) string {
	switch fe.Kind() {
	case reflect.String:
		return fe.Param() + " karakter"
	case reflect.Slice, reflect.Array, reflect.Map:
		return fe.Param() + " item"
	default:
		return fe.Param()
	}
}
