// Package kit provides utilities for handling HTTP request parsing and validation
// in Go applications. This file defines the ParseRequestBody function, which
// simplifies parsing incoming HTTP request bodies into structured Go objects
// and validates them using a provided Validator, ensuring robust input handling.

package kit

import (
	"errors"
	"mime"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Validator interface {
	StructTranslated(s interface{}) error
}

// ErrUnsupportedMediaType is returned (wrapped in a 415 HTTPError) when the
// request body is not JSON.
var ErrUnsupportedMediaType = errors.New("request body must be JSON")

// ParseRequestBody decodes a JSON request body into out and validates it.
// Only JSON bodies are accepted: form, multipart, and XML decoders bind
// exported fields by their Go name and would ignore json:"-" tags.
func ParseRequestBody(out any, c *fiber.Ctx, v Validator) error {
	if !isJSONContentType(string(c.Request().Header.ContentType())) {
		return HTTPUnsupportedMediaTypeError("unsupported-media-type", ErrUnsupportedMediaType)
	}

	if err := c.App().Config().JSONDecoder(c.Body(), out); err != nil {
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

// isJSONContentType reports whether contentType is application/json or a
// structured-syntax JSON type such as application/vnd.api+json.
func isJSONContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	_, subtype, ok := strings.Cut(mediaType, "/")
	if !ok {
		return false
	}
	return subtype == "json" || strings.HasSuffix(subtype, "+json")
}
