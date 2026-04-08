package response

import (
	"github.com/gofiber/fiber/v2"
)

// ErrorBody matches the centralized JSON error shape.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorEnvelope is the top-level error response.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// JSONError writes a structured error (apperror or generic).
func JSONError(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(ErrorEnvelope{
		Error: ErrorBody{Code: code, Message: message},
	})
}

// JSON writes success JSON.
func JSON(c *fiber.Ctx, status int, body any) error {
	return c.Status(status).JSON(body)
}
