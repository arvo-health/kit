package kit_test

import (
	"errors"
	"testing"

	"github.com/arvo-health/kit"
	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	tests := []struct {
		name              string
		errorCode         string
		format            string
		formatArgs        []any
		details           []string
		cause             error
		expectedString    string
		expectedError     string
		expectedCause     string
		expectedUnwrapped error
	}{
		{
			name:           "Simple error with no args",
			errorCode:      "SIMPLE_ERROR",
			format:         "this is a simple error",
			expectedString: "this is a simple error",
			expectedError:  "this is a simple error",
			expectedCause:  "",
		},
		{
			name:           "Error with formatting",
			errorCode:      "FORMATTED_ERROR",
			format:         "error %d: %s",
			formatArgs:     []any{404, "not found"},
			expectedString: "error 404: not found",
			expectedError:  "error 404: not found",
			expectedCause:  "",
		},
		{
			name:           "Error with details",
			errorCode:      "DETAILS_ERROR",
			format:         "some error",
			details:        []string{"detail1", "detail2"},
			expectedString: "some error",
			expectedError:  "some error",
			expectedCause:  "",
		},
		{
			name:              "Error with cause",
			errorCode:         "CAUSE_ERROR",
			format:            "outer error",
			cause:             errors.New("inner cause"),
			expectedString:    "outer error",
			expectedError:     "outer error: inner cause",
			expectedCause:     "inner cause",
			expectedUnwrapped: errors.New("inner cause"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := kit.NewErrorf(tt.errorCode, tt.format, tt.formatArgs...).
				WithDetails(tt.details).
				WrapCause(tt.cause)

			assert.Equal(t, tt.errorCode, err.Code())
			assert.Equal(t, tt.expectedString, err.String())
			assert.Equal(t, tt.expectedError, err.Error())
			assert.Equal(t, tt.expectedCause, err.Cause())
			assert.Equal(t, tt.expectedUnwrapped, errors.Unwrap(err))
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name         string
		sentinel     *kit.Error
		expectedCode string
		expectedMsg  string
	}{
		{
			name:         "Test ErrBadInput",
			sentinel:     kit.ErrBadInput,
			expectedCode: "BAD_INPUT",
			expectedMsg:  "Algo deu errado com os dados informados. Verifique e tente novamente.",
		},
		{
			name:         "Test ErrRequestValidation",
			sentinel:     kit.ErrRequestValidation,
			expectedCode: "REQUEST_VALIDATION",
			expectedMsg:  "Não foi possível concluir requisição. Verifique os dados informados e tente novamente.",
		},
		{
			name:         "Test ErrActionDenied",
			sentinel:     kit.ErrActionDenied,
			expectedCode: "ACTION_DENIED",
			expectedMsg:  "Você não tem permissão para realizar esta ação. Se precisar de acesso, contate o administrador.",
		},
		{
			name:         "Test ErrUnauthorized",
			sentinel:     kit.ErrUnauthorized,
			expectedCode: "UNAUTHORIZED",
			expectedMsg:  "Você não tem autorização para acessar este recurso.",
		},
	}

	// Loop through each sentinel error
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedCode, tt.sentinel.Code())
			assert.Equal(t, tt.expectedMsg, tt.sentinel.String())
		})
	}
}
