package middleware

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/stelgkio/affilflow/backend/pkg/apperror"
	"github.com/stelgkio/affilflow/backend/pkg/response"
)

// ErrorHandler is Fiber's centralized error handler.
func ErrorHandler(c *fiber.Ctx, err error) error {
	if e, ok := apperror.AsError(err); ok {
		status := fiber.StatusInternalServerError
		switch e.Kind {
		case apperror.KindInvalid:
			status = fiber.StatusBadRequest
		case apperror.KindNotFound:
			status = fiber.StatusNotFound
		case apperror.KindConflict:
			status = fiber.StatusConflict
		case apperror.KindUnauthorized:
			status = fiber.StatusUnauthorized
		case apperror.KindForbidden:
			status = fiber.StatusForbidden
		case apperror.KindInternal:
			status = fiber.StatusInternalServerError
		}
		return response.JSONError(c, status, e.Code, e.Message)
	}

	var fe *fiber.Error
	if errors.As(err, &fe) {
		return response.JSONError(c, fe.Code, "http_error", fe.Message)
	}

	return response.JSONError(c, fiber.StatusInternalServerError, "internal", "internal server error")
}
