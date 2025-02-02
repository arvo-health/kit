/*
Package validator provides utilities for struct validation in Go.
It wraps the `go-playground/validator` library, adding support for
error messages translated into Portuguese and offering structured
and standardized validation error handling.

Key Features:
- Simplified struct validation using tags.
- Customizable tag names for error messages with `custom` tag.
- Error messages with Portuguese translations.
- Structured validation errors with easily readable error keys.
*/
package validator

import (
	"errors"
	"maps"
	"reflect"
	"strings"

	"github.com/go-playground/locales/pt_BR"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	brTranslations "github.com/go-playground/validator/v10/translations/pt_BR"
)

// Validator is a wrapper around the `go-playground/validator` library.
// It provides struct validation and error translation into Portuguese.
type Validator struct {
	validate   *validator.Validate // The underlying validation engine.
	translator ut.Translator       // Translator for localized error messages.
}

// New initializes a new Validator instance with Portuguese translations.
// It configures the validator with custom tag name parsing for error messages.
func New() (*Validator, error) {
	ptBR := pt_BR.New()                    // Load the Brazilian Portuguese locale.
	uni := ut.New(ptBR)                    // Create a Universal Translator.
	trans, _ := uni.GetTranslator("pt_br") // Get the Portuguese translator.

	validate := validator.New() // Initialize the validator instance.

	// Customize tag names for error messages by extracting the `custom` tag
	// or falling back to the field name.
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := field.Tag.Get("custom")
		if name == "" {
			return field.Name
		}
		return name
	})

	// Register Portuguese translations for validation errors.
	if err := brTranslations.RegisterDefaultTranslations(validate, trans); err != nil {
		return nil, err
	}

	return &Validator{
		validate:   validate,
		translator: trans,
	}, nil
}

// Validate validates a struct and returns an *Error if any validation rules are violated.
// If validation is successful, it returns nil.
func (v Validator) Validate(s interface{}) *Error {
	err := v.validate.Struct(s)
	if err == nil {
		return nil // No validation errors.
	}

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		// Translate and sanitize validation error messages for readability.
		translations := validationErrors.Translate(v.translator)
		return &Error{
			message:     "validation failed",
			validations: sanitizeKeys(translations),
		}
	}

	// Return a generic error message if validation errors could not be translated.
	return &Error{
		message: "Unknown validation error",
	}
}

// sanitizeKeys simplifies validation error keys by removing struct prefixes.
// For example, `User.Email` becomes `Email` for better readability.
func sanitizeKeys(validationsErrs validator.ValidationErrorsTranslations) map[string]string {
	m := make(map[string]string, len(validationsErrs))
	for k := range maps.Keys(validationsErrs) {
		parts := strings.Split(k, ".")
		m[parts[len(parts)-1]] = validationsErrs[k]
	}
	return m
}
