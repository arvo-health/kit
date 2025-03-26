package kit_test

import (
	"testing"

	"github.com/arvo-health/kit"
	"github.com/stretchr/testify/assert"
)

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name             string
		message          string
		initialErrors    []string
		additionalErrors []string
		expectedErrors   []string
		expectHasErrors  bool
	}{
		{
			name:             "No validations initially",
			message:          "Validation failed",
			initialErrors:    nil,
			additionalErrors: nil,
			expectedErrors:   nil,
			expectHasErrors:  false,
		},
		{
			name:             "With initial validations",
			message:          "Validation failed",
			initialErrors:    []string{"Field A is required", "Field B must be a number"},
			additionalErrors: nil,
			expectedErrors:   []string{"Field A is required", "Field B must be a number"},
			expectHasErrors:  true,
		},
		{
			name:             "Add additional validations",
			message:          "Validation failed",
			initialErrors:    []string{"Field A is required"},
			additionalErrors: []string{"Field C cannot exceed 10 characters", "Field D must be unique"},
			expectedErrors:   []string{"Field A is required", "Field C cannot exceed 10 characters", "Field D must be unique"},
			expectHasErrors:  true,
		},
		{
			name:             "No validations after initialization but new were added",
			message:          "Validation failed",
			initialErrors:    []string{},
			additionalErrors: []string{"Field E is invalid"},
			expectedErrors:   []string{"Field E is invalid"},
			expectHasErrors:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize ValidationErrors
			validationErrors := kit.NewValidationErrors(tt.message, tt.initialErrors...)

			// Add additional validations if any
			if tt.additionalErrors != nil {
				validationErrors.Add(tt.additionalErrors...)
			}

			// Assert validations
			assert.Equal(t, tt.expectedErrors, validationErrors.Validations(), "Validations should match")

			// Check if there are validations
			assert.Equal(t, tt.expectHasErrors, validationErrors.HasValidations(), "HasValidations() should match the expected value")
			assert.Equal(t, !tt.expectHasErrors, validationErrors.HasNoValidations(), "HasNoValidations() should match the expected value")

			// Check error message
			assert.Equal(t, tt.message, validationErrors.Error(), "DomainError message should match")

			// Check ErrorOrNil behavior
			if tt.expectHasErrors {
				assert.NotNil(t, validationErrors.ErrorOrNil(), "ErrorOrNil should return the error instance when there are validations")
			} else {
				assert.Nil(t, validationErrors.ErrorOrNil(), "ErrorOrNil should return nil when there are no validations")
			}
		})
	}
}
