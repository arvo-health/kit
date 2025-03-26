package kit_test

import (
	"errors"
	"testing"

	"github.com/arvo-health/kit"
	"github.com/stretchr/testify/assert"
)

func TestNewResponseError(t *testing.T) {
	tests := []struct {
		name                   string
		inputStatus            int
		inputError             error
		expectedCode           string
		expectedStatus         int
		expectedMessage        string
		expectedCause          string
		expectedDetails        []string
		expectedUnwrappedError error
	}{
		{
			name:                   "Custom DomainError with Details",
			inputStatus:            400,
			inputError:             kit.NewDomainErrorf("CUSTOM_ERROR", "A custom %s occurred").WithArgs("error").WithDetails([]string{"detail1", "detail2"}),
			expectedCode:           "CUSTOM_ERROR",
			expectedStatus:         400,
			expectedMessage:        "A custom error occurred",
			expectedCause:          "",
			expectedDetails:        []string{"detail1", "detail2"},
			expectedUnwrappedError: kit.NewDomainErrorf("CUSTOM_ERROR", "A custom %s occurred").WithArgs("error").WithDetails([]string{"detail1", "detail2"}),
		},
		{
			name:                   "Validation DomainError",
			inputStatus:            422,
			inputError:             kit.NewValidationErrors("validation failed", []string{"field1 is invalid", "field2 is required"}...),
			expectedCode:           "VALIDATION",
			expectedStatus:         422,
			expectedMessage:        "validation failed",
			expectedCause:          "",
			expectedDetails:        []string{"field1 is invalid", "field2 is required"},
			expectedUnwrappedError: kit.NewValidationErrors("validation failed", []string{"field1 is invalid", "field2 is required"}...),
		},
		{
			name:                   "Unknown DomainError",
			inputStatus:            500,
			inputError:             errors.New("unexpected issue"),
			expectedCode:           "UNKNOWN",
			expectedStatus:         500,
			expectedMessage:        "Ocorreu um erro inesperado. Tente novamente mais tarde ou contate o administrador.",
			expectedCause:          "unexpected issue",
			expectedDetails:        nil,
			expectedUnwrappedError: errors.New("unexpected issue"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respErr := kit.NewResponseError(tt.inputStatus, tt.inputError)

			assert.Equal(t, tt.expectedCode, respErr.Code, "Code should match")
			assert.Equal(t, tt.expectedStatus, respErr.Status, "Status should match")
			assert.Equal(t, tt.expectedMessage, respErr.Message, "Message should match")
			assert.Equal(t, tt.expectedCause, respErr.Cause, "Cause should match")
			assert.Equal(t, tt.expectedDetails, respErr.Details, "Details should match")
			assert.Equal(t, tt.expectedUnwrappedError, tt.inputError, "DomainError should be the same")
		})
	}
}
