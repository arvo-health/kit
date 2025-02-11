// Package kit provides utilities for simplifying the creation and testing of HTTP handlers in Go applications.
// This file defines convenient types and functions for managing HTTP request and response structures,
// as well as assertion helpers for error testing in test cases.

package kit

import (
	"net/http"

	"github.com/stretchr/testify/assert"
)

// Map is a type alias for a map with string keys and values of any type, allowing flexible data representation.
type Map map[string]interface{}

// Request is a struct that represents the request body of the handler.
// It contains a map that can be used to marshal the request body into JSON format and a map for the headers.
// This struct is used to facilitate the creation and handling of HTTP requests
// in the test cases for the API handlers.
type Request struct {
	Body   Map
	Header http.Header
}

// Response is a struct that represents the response body of the handler.
// It contains a map that can be used to unmarshal the response body and a status code.
// This struct is used to facilitate the assertion of HTTP responses in the test cases
// for the API handlers. It is also used to represent the expected response from the handler.
type Response struct {
	Body       Map
	StatusCode int
}

// AssertErrorIs returns an assert.ErrorAssertionFunc that checks if the error is of the expected type.
// It uses the assert.ErrorIs function from the testify/assert package to perform the check.
// The function takes an expectedError and optional message arguments (msgAndArgs) to provide additional context in case of assertion failure.
func AssertErrorIs(expectedError error, msgAndArgs ...interface{}) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...interface{}) bool {
		return assert.ErrorIs(t, err, expectedError, msgAndArgs...)
	}
}
