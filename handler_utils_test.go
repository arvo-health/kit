package kit_test

import (
	"errors"
	"testing"

	"github.com/arvo-health/kit"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestParseRequestBody(t *testing.T) {
	type TestInput struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email"`
	}

	tests := []struct {
		name           string
		inputBody      string
		expectedOutput TestInput
		expectedError  any
	}{
		{
			name:           "valid input",
			inputBody:      `{"name":"John","email":"john@example.com"}`,
			expectedOutput: TestInput{Name: "John", Email: "john@example.com"},
			expectedError:  nil,
		},
		{
			name:           "invalid JSON format",
			inputBody:      `{"name":"John","email":"john@example.com"`,
			expectedOutput: TestInput{},
			expectedError:  kit.ErrBadInput,
		},
		{
			name:           "validation error",
			inputBody:      `{"name":"","email":"invalid email"}`,
			expectedOutput: TestInput{Name: "", Email: "invalid email"},
			expectedError:  kit.ErrRequestValidation,
		},
	}

	v := kit.NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a Fiber App and mock context
			app := fiber.New()
			c := app.AcquireCtx(&fasthttp.RequestCtx{})
			defer app.ReleaseCtx(c)

			// Set the request body
			c.Request().Header.SetContentType("application/json")
			c.Request().SetBody([]byte(tt.inputBody))

			var output TestInput
			err := kit.ParseRequestBody(&output, c, v)

			// Assert output and errors
			if tt.expectedError == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, output)
			} else {
				// Verify the returned error
				assert.True(t, errors.As(err, &tt.expectedError), "Error should be of type *kit.ValidationErrors")
				assert.Error(t, err)
			}
		})
	}
}
