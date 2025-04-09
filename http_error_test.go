package kit_test

import (
	"errors"
	"testing"

	"github.com/arvo-health/kit"
	"github.com/stretchr/testify/assert"
)

func TestNewHTTPError(t *testing.T) {
	tests := []struct {
		name            string
		inputStatus     int
		inputError      error
		inputSlug       string
		expectedSlug    string
		expectedStatus  int
		expectedMessage string
		expectedDetails []string
	}{
		{
			name:            "Custom DomainError with Details",
			inputStatus:     400,
			inputError:      kit.NewValidationErrors("A validation error occurred", "field1 is invalid", "field2 is required"),
			inputSlug:       "validation-error",
			expectedSlug:    "validation-error",
			expectedStatus:  400,
			expectedMessage: "A validation error occurred",
			expectedDetails: []string{"field1 is invalid", "field2 is required"},
		},
		{
			name:            "Standard error without details",
			inputStatus:     500,
			inputError:      errors.New("internal error happened"),
			inputSlug:       "",
			expectedSlug:    "unknown-error",
			expectedStatus:  500,
			expectedMessage: "internal error happened",
			expectedDetails: nil,
		},
		{
			name:            "No error provided (fallback to unknown error)",
			inputStatus:     500,
			inputError:      nil,
			inputSlug:       "unknown-slug",
			expectedSlug:    "unknown-slug",
			expectedStatus:  500,
			expectedMessage: "unknown error",
			expectedDetails: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpError := kit.NewHTTPError(tt.inputStatus, tt.inputSlug, tt.inputError)

			// Validações do HTTPError
			assert.Equal(t, tt.expectedStatus, httpError.Status, "HTTP status should match")
			assert.Equal(t, tt.expectedSlug, httpError.Slug, "Slug should match")
			assert.Equal(t, tt.expectedMessage, httpError.Message, "Error message should match")
			assert.Equal(t, tt.expectedDetails, httpError.Details, "Details should match (if any)")
		})
	}
}

func TestHTTPErrorString(t *testing.T) {
	tests := []struct {
		name           string
		httpError      *kit.HTTPError
		expectedString string
	}{
		{
			name: "HTTPError with Details",
			httpError: &kit.HTTPError{
				Slug:    "validation-error",
				Status:  400,
				Message: "Validation error occurred",
				Details: []string{"field1 is invalid", "field2 is required"},
			},
			expectedString: "[validation-error] Validation error occurred (field1 is invalid,field2 is required)",
		},
		{
			name: "HTTPError without Details",
			httpError: &kit.HTTPError{
				Slug:    "internal-error",
				Status:  500,
				Message: "Internal server error",
			},
			expectedString: "[internal-error] Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedString, tt.httpError.String(), "HTTPError string representation should match")
		})
	}
}

func TestPredefinedHTTPErrorFuncs(t *testing.T) {
	tests := []struct {
		name           string
		errorFunc      func(err error) *kit.HTTPError
		inputError     error
		expectedSlug   string
		expectedStatus int
	}{
		{
			name:           "HTTPInternalServerError creates error with proper fields",
			errorFunc:      kit.HTTPInternalServerError,
			inputError:     errors.New("internal issue"),
			expectedSlug:   "unknown-error",
			expectedStatus: 500,
		},
		{
			name:           "HTTPUnauthorizedError creates error with proper fields",
			errorFunc:      kit.HTTPUnauthorizedError,
			inputError:     errors.New("unauthorized access"),
			expectedSlug:   "unauthorized",
			expectedStatus: 401,
		},
		{
			name:           "HTTPBadRequestError creates error with proper fields",
			errorFunc:      func(err error) *kit.HTTPError { return kit.HTTPBadRequestError("bad-request", err) },
			inputError:     errors.New("bad input provided"),
			expectedSlug:   "bad-request",
			expectedStatus: 400,
		},
		{
			name:           "HTTPUnprocessableEntityError creates error with proper fields",
			errorFunc:      func(err error) *kit.HTTPError { return kit.HTTPUnprocessableEntityError("unprocessable-input", err) },
			inputError:     errors.New("validation failed"),
			expectedSlug:   "unprocessable-input",
			expectedStatus: 422,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpError := tt.errorFunc(tt.inputError)

			// Validações gerais de erro retornado
			assert.Equal(t, tt.expectedSlug, httpError.Slug, "Slug should match")
			assert.Equal(t, tt.expectedStatus, httpError.Status, "HTTP status should match")
			assert.Equal(t, tt.inputError.Error(), httpError.Message, "Error message should match")
		})
	}
}
