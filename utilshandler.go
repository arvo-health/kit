// Package kit provides utilities for handling HTTP request parsing and validation
// in Go applications. This file defines the ParseRequestBody function, which
// simplifies parsing incoming HTTP request bodies into structured Go objects
// and validates them using a provided Validator, ensuring robust input handling.

package kit

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// ParseRequestBody parses and validates the request body into the provided output struct, returning an error on failure.
// It uses the Fiber context to extract the body and the Validate struct for validation and error translation.
func ParseRequestBody(out interface{}, c *fiber.Ctx, v *Validate) error {
	// parse and validate the request body using the Fiber context
	if err := c.BodyParser(out); err != nil {
		err = ErrBadInput.WrapCause(err)
		return NewResponseError(http.StatusBadRequest, err)
	}

	// validate the parsed body using the provided Validator
	if err := v.StructTranslated(out); err != nil {
		var validationErrors *ValidationErrors
		if errors.As(err, &validationErrors) {
			err = ErrRequestValidation.WrapCause(err)
			return NewResponseError(http.StatusBadRequest, err)
		}
		return err
	}

	return nil
}
