# Kit

**Kit** is a foundational Go library designed to help developers build robust, consistent, and maintainable applications. It provides tools for structured logging, validation, error handling, and middleware integration for the **Fiber** web framework.

## Features

- **Validation**: Simplify struct validation with a centralized system that generates localized, detailed error messages using the `validator` package.
- **Error Handling**: Standardize application error responses with the `responseerror` package, enabling structured, reusable error handling.
- **Fiber Integration**: Seamlessly integrate error handling and logging into your Fiber-based applications.
- **Structured Logging**: Leverage JSON-based, context-rich logging with the `logger` package.
- **Middleware**: Add reusable middleware, like request logging, tailored for Fiber applications.

## Package Overview

```plaintext
kit/
├── logger/
│   ├── logger.go                   # Structured logging utilities
│   ├── middleware.go               # Logging middleware for Fiber
├── responseerror/
│   ├── fiber_error_handler.go      # Fiber-compatible error handler
│   ├── response_error.go           # Structured error responses
│   ├── response_error_registry.go  # Centralized error-to-response mapping
├── validator/
│   ├── validator.go                # Struct validation with localized messages
```

## Package Details

### Logger

The logger package provides:

- JSON-based structured logging using the slog library.
- Contextual attributes such as request IDs and user details for enhanced observability.
- Middleware integration for automatic request and response logging in Fiber.

### ResponseError

The responseerror package standardizes error handling by:

- Mapping domain-specific errors to structured HTTP responses.
- Supporting reusable error handlers for Fiber.

### Validator

The validator package wraps the go-playground/validator library to:

- Enable tag-based validation on structs.
- Generate error messages in Portuguese.
- Provide structured and simplified error responses.

## Usage Examples

### 1. Logging with Middleware

Use the `logger` package to create structured logs and integrate it with the `Middleware` for Fiber.

```go
import (
	"github.com/arvo-health/claim-mgmt/kit/logger"
    "github.com/gofiber/fiber/v2"
)

func main() {
    log := logger.New(slog.LevelInfo,
        slog.Group("service",
            slog.String("name", "service-name"), // retrieve from config
            slog.String("env", "prod"), // env=[prod,dev,hml,test]
            slog.String("version", "v1.0.0"),
        ),
    )

    errorRegistry := responseerror.NewRegistry()
    // Add error mappings to the registry

    app := fiber.New()

    app.Use(logger.Middleware(log, errorRegistry))

    app.Get("/", func (c *fiber.Ctx) error {
        return c.SendString("Hello, world!")
    })

    app.Listen(":8080")
}
```

### 2. Error Handling in Fiber

Use the `responseerror` package to centralize error handling in your Fiber app.

```go
import (
    "github.com/arvo-health/kit/responseerror"
    "github.com/gofiber/fiber/v2"
)

func main() {
    var ErrExample = errors.New("example error")
    registry := responseerror.NewRegistry().
        Add(ErrExample, "ERR-001", http.StatusUnprocessableEntity)

    app := fiber.New(fiber.Config{
        ErrorHandler: responseerror.FiberErrorHandler(registry),
    })

    app.Get("/", func (c *fiber.Ctx) error {
        return ErrExample
    })

    app.Listen(":8080")
}
```

### 3. Validation

Simplify struct validation with the `validator` package, which provides localized error messages.

```go
import (
    "errors"
    "fmt"
    "net/http"

    "github.com/arvo-health/kit/responseerror"
    "github.com/arvo-health/kit/validator"
    "github.com/gofiber/fiber/v2"
)

func main() {

    var ErrAnalysisValidation = errors.New("analysis validation failed")

    // Create a new error registry and add the custom error.
    registry := responseerror.NewRegistry().
        Add(ErrAnalysisValidation, "ERR-032", http.StatusBadRequest)

    // Create a custom error handler for Fiber.
    errorHandler := responseerror.FiberErrorHandler(registry)

    // Create a new Fiber app with the custom error handler.
    app := fiber.New(fiber.Config{
        ErrorHandler: errorHandler,
    })

    app.Get("/user", func(c *fiber.Ctx) error {
        type User struct {
            Name   string `validate:"required" custom:"Nome"`
            Age    int    `validate:"gte=18" custom:"Idade"`
            Gender string `validate:"required,oneof=M F" custom:"Gênero"`
            Email  string `validate:"email"`
        }

        // Create a new validator instance.
        validation, _ := validator.New()

        user := User{Email: "invalid-email", Gender: "X"}

        // Validate the user struct.
        err := validation.Validate(user)
        if err != nil {
            return err
        }

        return c.SendStatus(http.StatusOK)
    })

    // Complex validation example.
    app.Get("/analysis", func(c *fiber.Ctx) error {
        type Analysis struct {
            Status      string
            Description string
        }

        analysis := Analysis{Status: "DENIED"}

        // Create a new validator.Error instance.
        validatorErr := validator.NewError("analysis validation failed")

        // Do complex validation
        if analysis.Status == "DENIED" && len(analysis.Description) < 5 {
            validatorErr.AddValidation("Descrição", "Descrição deve ter pelo menos 5 caracteres")
        }

        // Do more complex validation
        if true {
            // Add more field-level validation error.
            validatorErr.AddValidation("Status", "Status deve ser xpto")
        }

        // Check if the validator.Error has any validations.
        if validatorErr.HasValidations() {
            // Wrap the validator.Error with the custom analysis validation error.
            return validatorErr.Wrap(ErrAnalysisValidation)
        }

        return c.SendStatus(http.StatusOK)
    })

    app.Listen(":8080")
}

```

The response error will be:

```json
// GET /user
// 422 Unprocessable Entity
{
  "error": {
    "code": "ERR-001",
    "message": "validation failed",
    "details": {
      "Email": "Email deve ser um endereço de e-mail válido",
      "Gênero": "Gênero deve ser um de [M F]",
      "Idade": "Idade deve ser 18 ou superior",
      "Nome": "Nome é um campo obrigatório"
    }
  }
}
```

```json
// GET /analysis
// 400 Bad Request
{
  "error": {
    "code": "ERR-032",
    "message": "analysis validation failed",
    "details": {
      "Descrição": "Descrição deve ter pelo menos 5 caracteres",
      "Status": "Status deve ser xpto"
    }
  }
}
```

## Contributions

Contributions and feedbacks are welcome!
