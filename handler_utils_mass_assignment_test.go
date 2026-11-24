package kit_test

import (
	"bytes"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arvo-health/kit"
	"github.com/gofiber/fiber/v2"
)

// accountRequest is the README ParseRequestBody shape, plus exported fields
// hidden from JSON clients. Those fields stand in for a role, an owner id,
// and a verified flag that a handler would persist after a successful parse.
type accountRequest struct {
	Name     string `json:"name" validate:"required"`
	Role     string `json:"-"`
	OwnerID  int    `json:"-"`
	Verified bool   `json:"-"`
}

// embeddedAccountRequest hides the same fields in an embedded struct.
// The form and XML decoders promote exported embedded fields.
type embeddedAccountRequest struct {
	Name string `json:"name" validate:"required"`
	embeddedPrivileges
}

type embeddedPrivileges struct {
	Role     string `json:"-"`
	OwnerID  int    `json:"-"`
	Verified bool   `json:"-"`
}

// TestParseRequestBodyRejectsNonJSONMassAssignment fails while ParseRequestBody
// accepts a form, multipart, or XML body and copies exported json:"-" fields
// into the struct a handler persists. It passes once those bodies are refused
// with 400 or 415 and the hidden fields stay at zero, including when refusal
// happens after a parse attempt.
func TestParseRequestBodyRejectsNonJSONMassAssignment(t *testing.T) {
	const (
		role     = "admin"
		ownerID  = 42
		verified = true
	)

	formBody := []byte("Name=Ada&Role=admin&OwnerID=42&Verified=true")
	formBodyLower := []byte("name=Ada&role=admin&ownerID=42&verified=true")
	xmlBody := []byte("<request><Name>Ada</Name><Role>admin</Role><OwnerID>42</OwnerID><Verified>true</Verified></request>")

	multipartBody, multipartType := multipartAccountBody(t, map[string]string{
		"Name":     "Ada",
		"Role":     "admin",
		"OwnerID":  "42",
		"Verified": "true",
	})
	multipartLowerBody, multipartLowerType := multipartAccountBody(t, map[string]string{
		"name":     "Ada",
		"role":     "admin",
		"ownerID":  "42",
		"verified": "true",
	})

	attacks := []struct {
		name        string
		contentType string
		body        []byte
	}{
		{name: "urlencoded Go field names", contentType: fiber.MIMEApplicationForm, body: formBody},
		{name: "urlencoded case-insensitive field names", contentType: fiber.MIMEApplicationForm, body: formBodyLower},
		{name: "urlencoded with charset", contentType: "application/x-www-form-urlencoded; charset=utf-8", body: formBody},
		{name: "multipart Go field names", contentType: multipartType, body: multipartBody},
		{name: "multipart case-insensitive field names", contentType: multipartLowerType, body: multipartLowerBody},
		{name: "application/xml", contentType: fiber.MIMEApplicationXML, body: xmlBody},
		{name: "text/xml", contentType: fiber.MIMETextXML, body: xmlBody},
		{name: "application/xml with charset", contentType: fiber.MIMEApplicationXMLCharsetUTF8, body: xmlBody},
	}

	for _, tc := range attacks {
		t.Run(tc.name, func(t *testing.T) {
			var parsed accountRequest
			status := postAccount(t, tc.contentType, tc.body, func(c *fiber.Ctx) (any, error) {
				parsed = accountRequest{}
				err := kit.ParseRequestBody(&parsed, c, kit.NewValidator())
				return parsed, err
			})
			assertHiddenFieldsRejected(t, status, parsed.Role, parsed.OwnerID, parsed.Verified, role, ownerID, verified)
		})

		t.Run(tc.name+" embedded", func(t *testing.T) {
			var parsed embeddedAccountRequest
			status := postAccount(t, tc.contentType, tc.body, func(c *fiber.Ctx) (any, error) {
				parsed = embeddedAccountRequest{}
				err := kit.ParseRequestBody(&parsed, c, kit.NewValidator())
				return parsed, err
			})
			assertHiddenFieldsRejected(t, status, parsed.Role, parsed.OwnerID, parsed.Verified, role, ownerID, verified)
		})

		t.Run(tc.name+" without required field", func(t *testing.T) {
			// Required Name is absent, so validation would already fail. The
			// hidden fields must still be unset when ParseRequestBody returns.
			body := bytes.ReplaceAll(tc.body, []byte("Name=Ada&"), nil)
			body = bytes.ReplaceAll(body, []byte("name=Ada&"), nil)
			body = bytes.ReplaceAll(body, []byte("<Name>Ada</Name>"), nil)
			if tc.contentType == multipartType || tc.contentType == multipartLowerType {
				fields := map[string]string{
					"Role": "admin", "OwnerID": "42", "Verified": "true",
				}
				if tc.contentType == multipartLowerType {
					fields = map[string]string{
						"role": "admin", "ownerID": "42", "verified": "true",
					}
				}
				var mt string
				body, mt = multipartAccountBody(t, fields)
				tcContentType := mt
				var parsed accountRequest
				status := postAccount(t, tcContentType, body, func(c *fiber.Ctx) (any, error) {
					parsed = accountRequest{}
					err := kit.ParseRequestBody(&parsed, c, kit.NewValidator())
					return parsed, err
				})
				assertHiddenFieldsRejected(t, status, parsed.Role, parsed.OwnerID, parsed.Verified, role, ownerID, verified)
				return
			}

			var parsed accountRequest
			status := postAccount(t, tc.contentType, body, func(c *fiber.Ctx) (any, error) {
				parsed = accountRequest{}
				err := kit.ParseRequestBody(&parsed, c, kit.NewValidator())
				return parsed, err
			})
			assertHiddenFieldsRejected(t, status, parsed.Role, parsed.OwnerID, parsed.Verified, role, ownerID, verified)
		})
	}
}

// TestParseRequestBodyJSONLeavesHiddenFieldsUnset is the JSON behavior the
// helper documents. A JSON client cannot set json:"-" fields, and a JSON
// body that carries the other required fields still parses.
func TestParseRequestBodyJSONLeavesHiddenFieldsUnset(t *testing.T) {
	body := []byte(`{"name":"Ada","Role":"admin","OwnerID":42,"Verified":true}`)
	cases := []struct {
		name        string
		contentType string
	}{
		{name: "application/json", contentType: fiber.MIMEApplicationJSON},
		{name: "application/json with charset", contentType: fiber.MIMEApplicationJSONCharsetUTF8},
		{name: "vendor json", contentType: "application/vnd.api+json"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var parsed accountRequest
			status := postAccount(t, tc.contentType, body, func(c *fiber.Ctx) (any, error) {
				parsed = accountRequest{}
				err := kit.ParseRequestBody(&parsed, c, kit.NewValidator())
				return parsed, err
			})
			if status != http.StatusCreated {
				t.Fatalf("status = %d, want %d", status, http.StatusCreated)
			}
			if parsed.Name != "Ada" {
				t.Fatalf("Name = %q, want Ada", parsed.Name)
			}
			if parsed.Role != "" || parsed.OwnerID != 0 || parsed.Verified {
				t.Fatalf("JSON body set hidden fields: %+v", parsed)
			}
		})

		t.Run(tc.name+" embedded", func(t *testing.T) {
			var parsed embeddedAccountRequest
			status := postAccount(t, tc.contentType, body, func(c *fiber.Ctx) (any, error) {
				parsed = embeddedAccountRequest{}
				err := kit.ParseRequestBody(&parsed, c, kit.NewValidator())
				return parsed, err
			})
			if status != http.StatusCreated {
				t.Fatalf("status = %d, want %d", status, http.StatusCreated)
			}
			if parsed.Name != "Ada" {
				t.Fatalf("Name = %q, want Ada", parsed.Name)
			}
			if parsed.Role != "" || parsed.OwnerID != 0 || parsed.Verified {
				t.Fatalf("JSON body set hidden fields: %+v", parsed)
			}
		})
	}
}

// postAccount runs the README handler shape: ParseRequestBody, and on success
// treat the struct as the record the handler would save.
func postAccount(t *testing.T, contentType string, body []byte, parse func(*fiber.Ctx) (any, error)) int {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: kit.ErrorHandler(logger)})
	var persisted any
	app.Post("/users", func(c *fiber.Ctx) error {
		parsed, err := parse(c)
		if err != nil {
			return err
		}
		persisted = parsed
		return c.Status(fiber.StatusCreated).JSON(persisted)
	})

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	req.Header.Set(fiber.HeaderContentType, contentType)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp.StatusCode
}

func assertHiddenFieldsRejected(t *testing.T, status int, role string, ownerID int, verified bool, wantRole string, wantOwnerID int, wantVerified bool) {
	t.Helper()
	if status != http.StatusBadRequest && status != http.StatusUnsupportedMediaType {
		t.Errorf("status = %d, want %d or %d", status, http.StatusBadRequest, http.StatusUnsupportedMediaType)
	}
	if role == wantRole || ownerID == wantOwnerID || verified == wantVerified {
		t.Errorf("hidden fields were bound (role=%q ownerID=%d verified=%v); they must stay unset even when the handler returns an error", role, ownerID, verified)
	}
	if role != "" || ownerID != 0 || verified {
		t.Errorf("hidden fields left non-zero (role=%q ownerID=%d verified=%v)", role, ownerID, verified)
	}
}

func multipartAccountBody(t *testing.T, fields map[string]string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for key, value := range fields {
		if err := w.WriteField(key, value); err != nil {
			t.Fatalf("WriteField(%s): %v", key, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	return buf.Bytes(), w.FormDataContentType()
}
