package kit_test

import (
	"testing"

	"github.com/arvo-health/kit"
	"github.com/stretchr/testify/assert"
)

// Struct to test validation logic.
type ExampleStruct struct {
	Name  string `validate:"required" custom:"Nome"`
	Email string `validate:"required,email" custom:"E-mail"`
	Age   int    `validate:"gte=18" custom:"Idade"`
}

func TestNewValidator(t *testing.T) {
	// Test that NewValidator initializes a Validate instance.
	validator := kit.NewValidator()
	assert.NotNil(t, validator, "Validator should not be nil")
	assert.NotNil(t, validator.Validate, "Validator.Validate should not be nil")
	assert.NotNil(t, validator.Translator, "Validator.Translator should not be nil")
}

func TestStructTranslated(t *testing.T) {
	tests := []struct {
		name           string
		input          ExampleStruct
		expectedErrors []string
		expectError    bool
	}{
		{
			name: "Valid struct",
			input: ExampleStruct{
				Name:  "John Doe",
				Email: "john.doe@example.com",
				Age:   25,
			},
			expectedErrors: nil,
			expectError:    false,
		},
		{
			name: "Missing required fields",
			input: ExampleStruct{
				Name:  "",
				Email: "",
				Age:   0,
			},
			expectedErrors: []string{
				"Nome é um campo obrigatório",
				"E-mail é um campo obrigatório",
				"Idade deve ser 18 ou superior",
			},
			expectError: true,
		},
		{
			name: "Invalid email and age below minimum",
			input: ExampleStruct{
				Name:  "Jane Doe",
				Email: "invalid-email",
				Age:   16,
			},
			expectedErrors: []string{
				"E-mail deve ser um endereço de e-mail válido",
				"Idade deve ser 18 ou superior",
			},
			expectError: true,
		},
	}

	validator := kit.NewValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.StructTranslated(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				// Verify the returned error messages and validations
				var validationErr *kit.ValidationErrors
				assert.ErrorAs(t, err, &validationErr, "DomainError should be of type *kit.ValidationErrors")
				assert.Equal(t, "validation failed", validationErr.Error(), "DomainError message should match")
				assert.ElementsMatch(t, tt.expectedErrors, validationErr.Validations(), "DomainError validations should match")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
