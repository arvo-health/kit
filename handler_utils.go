// Package kit provides utilities for handling HTTP request parsing and validation
// in Go applications. This file defines the ParseRequestBody function, which
// simplifies parsing incoming HTTP request bodies into structured Go objects
// and validates them using a provided Validator, ensuring robust input handling.

package kit

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

type Validator interface {
	StructTranslated(s interface{}) error
}

func ParseRequestBody(out any, c *fiber.Ctx, v Validator) error {
	// parse and validate the request body using the Fiber context
	if err := c.BodyParser(out); err != nil {
		return HTTPBadRequestError("bad-input", err)
	}

	// validate the parsed body using the provided Validator
	if err := v.StructTranslated(out); err != nil {
		var validationErrors *ValidationErrors
		if errors.As(err, &validationErrors) {
			return HTTPBadRequestError("request-validation", err)
		}
		return err
	}

	return nil
}
