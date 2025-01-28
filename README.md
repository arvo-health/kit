# Kit

**Kit** is a foundational Go library designed to help developers build robust, consistent, and maintainable applications. It provides tools for structured logging, validation, error handling, and middleware integration for the **Fiber** web framework.

---

## Features

- **Validation**: Simplify struct validation with a centralized system that generates localized, detailed error messages using the `validator` package.
- **Error Handling**: Standardize application error responses with the `responseerror` package, enabling structured, reusable error handling.
- **Fiber Integration**: Seamlessly integrate error handling and logging into your Fiber-based applications.

---

## Package Overview

```plaintext
kit/
├── responseerror/
│   ├── fiber_error_handler.go      # Fiber-compatible error handler
│   ├── response_error.go           # Structured error responses
│   ├── response_error_registry.go  # Centralized error-to-response mapping
├── validator/
│   ├── validator.go                # Struct validation with localized messages
```

## Package Details

### ResponseError

The responseerror package standardizes error handling by:

- Mapping domain-specific errors to structured HTTP responses.
- Providing categories for errors (e.g., validation, authentication, internal server errors).
- Supporting reusable error handlers for Fiber.

### Validator

The validator package wraps the go-playground/validator library to:

- Enable tag-based validation on structs.
- Generate error messages in Portuguese.
- Provide structured and simplified error responses.

---

## Usage Examples

### 1. Error Handling in Fiber

Use the `responseerror` package to centralize error handling in your Fiber app.

```go
import (
    "github.com/arvo-health/kit/responseerror"
    "github.com/gofiber/fiber/v2"
)

func main() {
  var ErrExample = errors.New("example error")
  registry := responseerror.NewRegistry().
    Add(ErrExample, "ERR-V001", http.StatusUnprocessableEntity)

  app := fiber.New(fiber.Config{
    ErrorHandler: responseerror.FiberErrorHandler(registry),
  })

  app.Get("/", func (c *fiber.Ctx) error {
    return ErrExample
  })

  app.Listen(":8080")
}
```

### 2. Validation

Simplify struct validation with the `validator` package, which provides localized error messages.

```go
import "github.com/arvo-health/kit/validator"

func main() {
  validator := validator.NewValidator()

  type Input struct {
    Name  string `validate:"required" custom:"Nome"`
    Age   int    `validate:"gte=18" custom:"Idade"`
    Email string `validate:"required,email"`
  }

  input := Input{}
  err := validator.Validate(input)
  if err != nil {
    fmt.Println(err.Validations()) // Outputs field-level error details.
  }
}
```

---

### Contributions

Contributions and feedbacks are welcome!
