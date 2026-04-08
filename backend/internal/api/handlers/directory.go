package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/middleware"
	"github.com/stelgkio/affilflow/backend/pkg/response"
)

// DirectoryPrograms GET /api/v1/directory/programs
func (h *Handlers) DirectoryPrograms(c *fiber.Ctx) error {
	list, err := h.Org.ListDiscoverable(c.UserContext())
	if err != nil {
		return err
	}
	return response.JSON(c, fiber.StatusOK, list)
}

// DirectoryApply POST /api/v1/directory/programs/:orgId/apply
func (h *Handlers) DirectoryApply(c *fiber.Ctx) error {
	orgID, err := uuid.Parse(c.Params("orgId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid organization id")
	}
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing user")
	}
	var body struct {
		Email *string `json:"email"`
	}
	_ = c.BodyParser(&body)
	if err := h.Discovery.Apply(c.UserContext(), orgID, uid, body.Email); err != nil {
		msg := err.Error()
		status := fiber.StatusBadRequest
		code := "apply_failed"
		if strings.Contains(msg, "invite-only") || strings.Contains(msg, "not discoverable") {
			status = fiber.StatusForbidden
			code = "apply_not_allowed"
		}
		if strings.Contains(msg, "already an affiliate") {
			status = fiber.StatusConflict
			code = "already_affiliate"
		}
		if strings.Contains(msg, "at capacity") {
			status = fiber.StatusConflict
			code = "program_full"
		}
		return response.JSONError(c, status, code, msg)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
