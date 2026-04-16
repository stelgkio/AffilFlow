package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/middleware"
	"github.com/stelgkio/affilflow/backend/pkg/response"
)

// InviteCreate POST /api/v1/campains/:campainId/invites
func (h *Handlers) InviteCreate(c *fiber.Ctx) error {
	campainID, err := uuid.Parse(c.Params("campainId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid campain id")
	}
	var body struct {
		Email *string `json:"email"`
	}
	_ = c.BodyParser(&body)
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	var createdBy *string
	if uid != "" {
		createdBy = &uid
	}
	plain, id, err := h.Invite.Create(c.UserContext(), campainID, body.Email, createdBy)
	if err != nil {
		code := "invite_failed"
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "invite limit") {
			code = "invite_limit"
			status = fiber.StatusConflict
		}
		return response.JSONError(c, status, code, err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, fiber.Map{
		"invite_id":    id,
		"invite_url":   h.Invite.JoinURL(plain),
		"email_queued": false,
	})
}

// InviteValidate GET /api/v1/invites/:token/validate
func (h *Handlers) InviteValidate(c *fiber.Ctx) error {
	token := strings.TrimSpace(c.Params("token"))
	if token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "token required")
	}
	inv, err := h.Invite.GetPendingInvite(c.UserContext(), token)
	if err != nil {
		return err
	}
	if inv == nil {
		return response.JSONError(c, fiber.StatusNotFound, "invalid_invite", "invalid or expired invite")
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"valid":           true,
		"campain_id": inv.CampainID,
		"expires_at":      inv.ExpiresAt,
	})
}

// InviteAccept POST /api/v1/invites/:token/accept
func (h *Handlers) InviteAccept(c *fiber.Ctx) error {
	token := strings.TrimSpace(c.Params("token"))
	if token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "token required")
	}
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing user")
	}
	var body struct {
		Email *string `json:"email"`
	}
	_ = c.BodyParser(&body)
	if err := h.Invite.Accept(c.UserContext(), token, uid, body.Email); err != nil {
		return response.JSONError(c, fiber.StatusBadRequest, "accept_failed", err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}
