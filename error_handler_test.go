package kit_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arvo-health/kit"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorHandler(t *testing.T) {
	tests := []struct {
		name              string
		inputError        error
		expectedStatus    int
		expectedErrorBody map[string]any
	}{
		{
			name:           "Fiber Error handled correctly",
			inputError:     fiber.NewError(http.StatusBadRequest, "bad request"),
			expectedStatus: http.StatusBadRequest,
			expectedErrorBody: map[string]any{
				"error": map[string]any{
					"code":        "fiber-err",
					"message":     "bad requestbad request",
					"status_code": float64(http.StatusBadRequest),
				},
			},
		},
		{
			name:           "ResponseError with ValidationError handled correctly",
			inputError:     kit.HTTPUnprocessableEntityError("validation", kit.NewValidationErrors("validation error", "field1 is required")),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedErrorBody: map[string]any{
				"error": map[string]any{
					"code":        "validation",
					"status_code": float64(http.StatusUnprocessableEntity),
					"message":     "validation error",
					"details":     []any{"field1 is required"},
				},
			},
		},
		{
			name:           "ResponseError with custom DomainError handled correctly",
			inputError:     kit.NewHTTPError(http.StatusConflict, "some-code", errors.New("any error message")),
			expectedStatus: http.StatusConflict,
			expectedErrorBody: map[string]any{
				"error": map[string]any{
					"code":        "some-code",
					"status_code": float64(http.StatusConflict),
					"message":     "any error message",
				},
			},
		},
		{
			name:           "Non-ResponseError handled as UNKNOWN",
			inputError:     errors.New("internal server error"),
			expectedStatus: http.StatusInternalServerError,
			expectedErrorBody: map[string]any{
				"error": map[string]any{
					"code":        "unknown-error",
					"status_code": float64(http.StatusInternalServerError),
					"message":     "internal server error",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Configuração do app Fiber
			app := fiber.New(fiber.Config{
				ErrorHandler: kit.ErrorHandler(),
			})

			// Rota de teste que retorna um erro
			app.Get("/test", func(c *fiber.Ctx) error {
				return tt.inputError
			})

			// Fazendo a requisição de teste
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Obter a resposta
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Parse do corpo da resposta
			var respBody map[string]any
			err = json.NewDecoder(resp.Body).Decode(&respBody)
			require.NoError(t, err)

			// Valida status e corpo do erro
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Equal(t, tt.expectedErrorBody["error"], respBody["error"])
		})
	}
}
