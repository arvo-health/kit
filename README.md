# Kit

**Kit** is a foundational Go library designed to help developers build robust, consistent, and maintainable applications. It provides tools for structured logging, validation, error handling, and middleware integration for the **Fiber** web framework.

---

## Features

- **Validation**: Simplify struct validation with a centralized system that generates localized, detailed error messages using the `validator` package.

---

## Package Overview

```plaintext
kit/
├── validator/
│   ├── validator.go                # Struct validation with localized messages
``` 

## Package Details

### Validator

The validator package wraps the go-playground/validator library to:

- Enable tag-based validation on structs.
- Generate error messages in Portuguese.
- Provide structured and simplified error responses.

---

## Usage Examples

### 1. Validation

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
