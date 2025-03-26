package kit_test

//
//import (
//	"encoding/json"
//	"errors"
//	"net/http"
//	"net/http/httptest"
//	"testing"
//
//	"github.com/arvo-health/kit"
//	"github.com/gofiber/fiber/v2"
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//)
//
//func TestErrorHandler(t *testing.T) {
//	tests := []struct {
//		name              string
//		inputError        error
//		expectedStatus    int
//		expectedErrorBody map[string]any
//	}{
//		{
//			name:           "ResponseError passed to handler",
//			inputError:     kit.HTTPBadRequestError( "bad-request", errors.New("bad request")),
//			expectedStatus: http.StatusBadRequest,
//			expectedErrorBody: map[string]any{
//				"error": map[string]any{
//					"cause":       "bad request",
//					"status_code": float64(400),
//					"message":     "Ocorreu um erro inesperado. Tente novamente mais tarde ou contate o administrador.",
//					"code":        "UNKNOWN",
//				},
//			},
//		},
//		{
//			name: "ResponseError with ValidationError passed to handler",
//			inputError: kit.HTTPUnprocessableEntityError("validation", kit.NewValidationErrors("validation error", "field1 is required")),
//			expectedStatus: http.StatusUnprocessableEntity,
//			expectedErrorBody: map[string]any{
//				"error": map[string]any{
//					"code":        "VALIDATION",
//					"status_code": float64(422),
//					"message":     "validation error",
//					"details":     []any{"field1 is required"},
//				},
//			},
//		},
//		{
//			name: "ResponseError with custom DomainError passed to handler",
//			inputError: kit.NewResponseError(http.StatusConflict,
//				kit.NewDomainErrorf("SOME_CODE", "any %s message", "error")),
//			expectedStatus: http.StatusConflict,
//			expectedErrorBody: map[string]any{
//				"error": map[string]any{
//					"code":        "SOME_CODE",
//					"status_code": float64(409),
//					"message":     "any error message",
//				},
//			},
//		},
//		{
//			name:           "Non-ResponseError handled as UNKNOWN",
//			inputError:     errors.New("internal server error"),
//			expectedStatus: http.StatusInternalServerError,
//			expectedErrorBody: map[string]any{
//				"error": map[string]any{
//					"code":        "UNKNOWN",
//					"status_code": float64(500),
//					"message":     "Ocorreu um erro inesperado. Tente novamente mais tarde ou contate o administrador.",
//					"cause":       "internal server error",
//				},
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Fiber app setup
//			app := fiber.New(fiber.Config{
//				ErrorHandler: kit.ErrorHandler(),
//			})
//
//			app.Get("/test", func(c *fiber.Ctx) error {
//				return tt.inputError
//			})
//
//			req := httptest.NewRequest(http.MethodGet, "/test", nil)
//
//			resp, err := app.Test(req)
//			require.NoError(t, err)
//			defer resp.Body.Close()
//
//			var respBody map[string]any
//			err = json.NewDecoder(resp.Body).Decode(&respBody)
//			require.NoError(t, err)
//
//			// Validate status code and error body
//			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
//			assert.Equal(t, tt.expectedErrorBody["error"], respBody["error"])
//		})
//	}
//}
