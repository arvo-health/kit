// Package kit provides struct validation utilities using `go-playground/validator`.
// This file defines a validation wrapper with support for localized (pt_BR) error messages,
// including initialization of a universal translator and custom error message handling.

package kit

import (
	"errors"
	"reflect"

	"github.com/go-playground/locales/pt_BR"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	brTranslations "github.com/go-playground/validator/v10/translations/pt_BR"
)

// Validate is a struct that wraps a validator instance and a translator for pt_BR localized error messages.
type Validate struct {
	*validator.Validate // The underlying validation engine.
	ut.Translator       // Translator for localized error messages.
}

// NewValidator initializes and returns a new Validate struct with a pt_BR translator and a custom tag name function.
func NewValidator() *Validate {
	ptBR := pt_BR.New()
	uni := ut.New(ptBR)
	trans, _ := uni.GetTranslator("pt_br")

	validate := validator.New(validator.WithRequiredStructEnabled()) // Initialize the validator instance.

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
	_ = brTranslations.RegisterDefaultTranslations(validate, trans)

	return &Validate{
		Validate:   validate,
		Translator: trans,
	}
}

// StructTranslated validates the given struct and returns translated validation error messages if any validation fails.
func (v *Validate) StructTranslated(s interface{}) error {
	err := v.Struct(s)
	if err == nil {
		return nil // No validation errors.
	}

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		validations := make([]string, 0, len(validationErrors))
		translations := validationErrors.Translate(v.Translator)
		for _, t := range translations {
			validations = append(validations, t)
		}
		return NewValidationErrors("validation failed", validations...)
	}
	return err
}
