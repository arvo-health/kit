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
	// Estrutura para teste
	type TestInput struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email,omitempty"`
	}

	// Casos de teste
	tests := []struct {
		name           string
		inputBody      string
		expectedOutput TestInput
		expectedError  error
	}{
		{
			name:           "Valid input",
			inputBody:      `{"name":"John","email":"john@example.com"}`,
			expectedOutput: TestInput{Name: "John", Email: "john@example.com"},
			expectedError:  nil,
		},
		{
			name:           "Invalid JSON format",
			inputBody:      `{"name":"John","email":"john@example.com"`, // JSON inválido
			expectedOutput: TestInput{},
			expectedError:  kit.HTTPBadRequestError("bad-input", errors.New("unexpected end of JSON input")),
		},
		{
			name:           "Validation error",
			inputBody:      `{"name":"","email":"invalid email"}`, // Campo "name" vazio
			expectedOutput: TestInput{Name: "", Email: "invalid email"},
			expectedError:  kit.HTTPBadRequestError("request-validation", errors.New("validation failed")),
		},
	}

	v := kit.NewValidator()

	// Executa cada caso de teste
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Configura o Fiber App e mock do context
			app := fiber.New()
			c := app.AcquireCtx(&fasthttp.RequestCtx{})
			defer app.ReleaseCtx(c)

			// Define o corpo da requisição
			c.Request().Header.SetContentType("application/json")
			c.Request().SetBody([]byte(tt.inputBody))

			var output TestInput
			err := kit.ParseRequestBody(&output, c, v)

			// Verifica a saída e os erros
			if tt.expectedError == nil {
				assert.NoError(t, err)                     // Sem erro esperado
				assert.Equal(t, tt.expectedOutput, output) // Verifica os dados do corpo
			} else {
				// Valida o erro retornado
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error()) // Verifica se o erro contém a mensagem esperada
			}
		})
	}
}
