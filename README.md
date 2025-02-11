Package kit is a foundational Go library designed to help developers build consistent, and maintainable
applications. It provides utilities for structured logging, validation, error handling, request parsing,
and middleware integration with the Fiber web framework.

# Key Features:

  - **Validation**: Provides a wrapper around `go-playground/validator`, supporting custom validation tags
    and localized error messages in Portuguese.

  - **Error Handling**: Standardizes application errors with structured responses through `ResponseError`.
    Includes a Fiber-compatible error handler (`ErrorHandler`) for seamless error processing.
    Provides structured error types (`Error` and `ValidationErrors`) for better debugging.

  - **Logging**: Leverages `slog` for structured, JSON-based logging with support for contextual attributes
    like request ID, user details, and response status. Includes `LoggerMiddleware` for request-based logging.

  - **Middleware**: Provides reusable middleware for Fiber, including `LoggerMiddleware` for logging request
    and response data with contextual information.

  - **Request Handling**: Simplifies HTTP request parsing and validation through `ParseRequestBody`,
    reducing boilerplate code for handling and validating request payloads.

  - **Testing Utilities**: Provides helper functions and types (`Map`, `Request`, `Response`) to facilitate
    the creation and testing of HTTP handlers.

  - **Context Management**: Defines `ContextKey` constants to facilitate storing and retrieving metadata
    (e.g., user info, logging context).

# Package Structure:

```plaintext
kit/
├── context_key.go         # Defines context keys for storing metadata
├── error.go               # Custom error types with error chaining support
├── error_handler.go       # Custom Fiber error handler for structured error responses
├── error_response.go      # Standardized error response structure
├── logger.go              # Structured logging utilities using slog
├── logger_middleware.go   # Middleware for request logging in Fiber
├── validator.go           # Wrapper for struct validation with localized messages
├── validator_error.go     # Custom validation error structure
├── utilshandler.go        # Utility functions for handling HTTP requests and validation
├── utilstest.go           # Testing utilities for HTTP requests and responses
├── doc.go                 # General package documentation
```

# Basic Usage Examples

## Error Handling, Custom Errors and Logging with Middleware

Use the `kit.LoggerMiddleware(logger)` to log structured request and response lod data. Use `kitNewErrorf()` to domain
custom errors and `kit.NewResponseError()` along with `kit.ErrorHandler()` for centralized error handling and consistent response body.

```go
package main

import (
    "log/slog"
    "net/http"

    "github.com/arvo-health/kit"
    "github.com/gofiber/fiber/v2"
)

// kit custom Error error
var ErrUserNotFound = kit.NewErrorf("USER_NOT_FOUND", "user with ID %s not found")

func main() {
    // kit logger example
    logger := kit.NewLogger(slog.LevelInfo,
        slog.Group("service",
            slog.String("name", "my-service"),
            slog.String("env", "prod"),
            slog.String("version", "v1.0.0"),
        ),
    )

    app := fiber.New(fiber.Config{
        // kit error handler example
        ErrorHandler: kit.ErrorHandler(),
    })
    // kit logger middleware example
    app.Use(kit.LoggerMiddleware(logger))

    app.Get("/", func(c *fiber.Ctx) error {
        return c.SendString("Hello, world!")
    }).Name("Hello world")

    app.Get("/users/:id", func(c *fiber.Ctx) error {
        id := c.Params("id")
        // kit response error with status 400
        return kit.NewResponseError(http.StatusNotFound,
            // kit custom Error error adding args
            ErrUserNotFound.WithArgs(id))
    }).Name("Get user by ID")

    app.Listen(":8080")
}
```
### Expected error responses and logs

#### Success log


```shell
  curl -X GET localhost:8080
```

```json
// log: request completed without error
{"time":"2025-02-03T18:12:36.089856-03:00","level":"INFO","msg":"Hello world: request completed","service":{"name":"my-service","env":"prod","version":"v1.0.0"},"request_id":"7cdbae25-9b3c-4d43-a2f7-9e94f51cafef","duration_ms":0,"user":{"email":"unknown","company":"unknown","company_category":"unknown","permissions":"<nil>"},"request":{"start_time":"2025-02-10T21:12:36.089776Z","method":"GET","route":"/","params":{},"queries":{}},"response":{"end_time":"2025-02-10T21:12:36.089825Z","status":200}}
```

#### Error response and log

```shell
  curl -X GET localhost:8080/users/203
```

```json
// response: 404 Not Found
{
  "error": {
    "code": "USER_NOT_FOUND",
    "status_code": 404,
    "message": "user with ID 203 not found"
  }
}
```

```json
// log: request completed with error, has error group
{"time":"2025-02-03T16:29:12.709121-03:00","level":"ERROR","msg":"Get user by ID: user with ID 203 not found","service":{"name":"my-service","env":"prod","version":"v1.0.0"},"request_id":"98f7d221-0a8a-4fc5-826f-fb7840d408f5","duration_ms":0,"error":{"code":"USER_NOT_FOUND","message":"user with ID 203 not found","cause":"","details":null},"user":{"email":"unknown","company":"unknown","company_category":"unknown","permissions":"<nil>"},"request":{"start_time":"2025-02-10T19:29:12.708577Z","method":"GET","route":"/user/:id","params":{"id":"203"},"queries":{}},"response":{"end_time":"2025-02-10T19:29:12.708673Z","status":404}}
```

## Validation and ValidationErrors with Error Handling

Use the  and `kit.NewValidator()` and `kit.NewValidationErrors()` for struct and business rule validation.
See the official [go-playground/validator](https://pkg.go.dev/github.com/go-playground/validator/v10#section-readme)
documentation for more tag validation options.

```go
package main

import (
    "net/http"
    "strings"

    "github.com/arvo-health/kit"
    "github.com/gofiber/fiber/v2"
)

// kit custom Error error
var ErrInsertUserValidation = kit.NewErrorf("USER_VALIDATION", "insert user validation failed")
var ErrInsertUserBusinessValidation = kit.NewErrorf("USER_BUSINESS_VALIDATION", "insert user bisiness validation failed")

func main() {
    app := fiber.New(fiber.Config{
        // kit error handler example
        ErrorHandler: kit.ErrorHandler(),
    })

    app.Post("/users", func(c *fiber.Ctx) error {

        type request struct {
            Name  string `validate:"required" custom:"Nome"`
            Email string `validate:"required" custom:"E-mail"`
        }

        // kit validator creation
        validator := kit.NewValidator()

        var req request
        // kit request body parsing and validation
        if err := kit.ParseRequestBody(&req, c, validator); err != nil {
            // it returns BadRequest kit.ResponseError (ErrBadInput or ErrRequestValidation)
            return err
        }

        type User struct {
            Name  string `validate:"gte=5" custom:"Nome"`
            Email string `validate:"email" custom:"E-mail"`
        }

        user := User{
            Name:  req.Name,
            Email: req.Email,
        }

        // kit struct validation and pt_BR translation
        if err := validator.StructTranslated(user); err != nil {
            // update de domain kit.Errorf custom error wrapping the error cause
            err = ErrInsertUserValidation.WrapCause(err)
            // then return UnprocessableEntity kit.ResponseError with the validations details
            return kit.NewResponseError(http.StatusUnprocessableEntity, err)
        }

        // create a custom kit.NewValidationErrors to perform some "complex" business validations
        validationErr := kit.NewValidationErrors("validation error")

        // perform some "complex" business validations
        if len(user.Name) < 10 && !strings.HasPrefix(user.Email, strings.ToLower(user.Name)) {
            // add a validation detail if the condition is not met
            validationErr.Add("E-mail deve ser igual ao Nome")
        }
        if !strings.HasSuffix(user.Email, "@arvo.com.br") {
            validationErr.Add("E-mail deve ter o domínio @arvo.com.br")
        }

        // then through the `.ErrorOrNil()` method, the error (if any) can be returned or checked and handled
        if err := validationErr.ErrorOrNil(); err != nil {
            // update de domain kit.Errorf custom error wrapping the error cause
            err = ErrInsertUserBusinessValidation.WrapCause(err)
            // then return UnprocessableEntity kit.ResponseError with the validations details
            return kit.NewResponseError(http.StatusUnprocessableEntity, err)
        }

        return c.SendStatus(http.StatusOK)
    })

    app.Listen(":8080")
}
```

### Expected error responses

#### Malformed JSON

```shell
  curl -X POST localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"malformed": json,'
```

```json
// response: 400 Bad Request
{
  "error": {
    "code": "BAD_INPUT",
    "status_code": 400,
    "message": "bad input",
    "cause": "unexpected end of JSON input"
  }
}
```

#### Request required fields validation

```shell
  curl -X POST localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Bruno"}'
```

```json
// response: 400 Bad Request
{
  "error": {
    "code": "REQUEST_VALIDATION",
    "status_code": 400,
    "message": "request validation failed",
    "cause": "validation failed",
    "details": [
      "E-mail é um campo obrigatório"
    ]
  }
}
```

The field `details` only exist when the error is a **validation** error.

#### Business struct validation

```shell
  curl -X POST localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Bru","email":"invalidarvo.com.br"}'
```

```json
// response: 422 Unprocessable Entity
{
  "error": {
    "code": "USER_VALIDATION",
    "status_code": 422,
    "message": "insert user validation failed",
    "cause": "validation failed",
    "details": [
      "Nome deve ter pelo menos 5 caracteres",
      "E-mail deve ser um endereço de e-mail válido"
    ]
  }
}
```

#### "Complex" business validation

```shell
  curl -X POST localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Bruno","email":"mars@email.com"}'
```

```json
// response: 422 Unprocessable Entity
{
  "error": {
    "code": "USER_BUSINESS_VALIDATION",
    "status_code": 422,
    "message": "insert user bisiness validation failed",
    "cause": "validation error",
    "details": [
      "E-mail deve ser igual ao Nome",
      "E-mail deve ter o domínio @arvo.com.br"
    ]
  }
}
```

## Installation

To install the package, run:

```shell
  go get github.com/arvo-health/kit
```

Then, import it in your Go application:

```go
import "github.com/arvo-health/kit"
```

# Contribution

Contributions and feedback are welcome!
