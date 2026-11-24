// Package kit provides utilities for handling HTTP request parsing and validation
// in Go applications. This file defines the ParseRequestBody function, which
// simplifies parsing incoming HTTP request bodies into structured Go objects
// and validates them using a provided Validator, ensuring robust input handling.

package kit

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

type Validator interface {
	StructTranslated(s interface{}) error
}

func ParseRequestBody(out any, c *fiber.Ctx, v Validator) error {
	if err := decodeRequestBody(out, c); err != nil {
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

// decodeRequestBody decodes JSON and XML bodies and refuses every other
// content type. Form and multipart bodies must never reach Fiber's schema
// decoder, which allocates slices sized by indexes taken from field names.
func decodeRequestBody(out any, c *fiber.Ctx) error {
	ctype := utils.ToLower(string(c.Request().Header.ContentType()))
	ctype = utils.ParseVendorSpecificContentType(ctype)
	if i := strings.IndexByte(ctype, ';'); i != -1 {
		ctype = ctype[:i]
	}
	ctype = strings.TrimSpace(ctype)

	switch {
	case strings.HasSuffix(ctype, "json"):
		return c.App().Config().JSONDecoder(c.Body(), out)
	case ctype == fiber.MIMETextXML || ctype == fiber.MIMEApplicationXML:
		if err := xml.Unmarshal(c.Body(), out); err != nil {
			return fmt.Errorf("failed to unmarshal: %w", err)
		}
		return nil
	default:
		return fiber.ErrUnsupportedMediaType
	}
}
