package kit_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/arvo-health/kit"
	"github.com/stretchr/testify/assert"
)

func TestAssertErrorIs(t *testing.T) {
	customError := errors.New("test error")
	wrappedError := fmt.Errorf("wrapped: %w", customError)

	tests := []struct {
		name           string
		givenError     error
		expectedError  error
		expectMatching bool
	}{
		{
			name:           "Matching DomainError",
			givenError:     customError,
			expectedError:  customError,
			expectMatching: true,
		},
		{
			name:           "Wrapped DomainError Matching",
			givenError:     wrappedError,
			expectedError:  customError,
			expectMatching: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isMatching := kit.AssertErrorIs(tt.expectedError)(t, tt.givenError)
			assert.Equal(t, tt.expectMatching, isMatching, "DomainError match assertion result mismatch")
		})
	}
}
