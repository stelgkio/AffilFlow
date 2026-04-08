package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/stelgkio/affilflow/backend/internal/auth"
	"github.com/stelgkio/affilflow/backend/internal/middleware"
	"github.com/stelgkio/affilflow/backend/pkg/response"
)

type onboardCompanyBody struct {
	Name string `json:"name"`
}

// OnboardCompany POST /api/v1/onboarding/company — create merchant org and promote user to admin.
func (h *Handlers) OnboardCompany(c *fiber.Ctx) error {
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing user")
	}
	var body onboardCompanyBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	ctx := c.UserContext()
	u, err := h.User.GetByID(ctx, uid)
	if err != nil || u == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "user not found")
	}
	if u.OrganizationID != nil {
		return fiber.NewError(fiber.StatusConflict, "user already belongs to an organization")
	}

	orgID, err := h.Org.Create(ctx, name)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if h.Sub != nil {
		if err := h.Sub.CreateFree(ctx, orgID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
	if err := h.User.SetOrganizationAndRole(ctx, uid, &orgID, "admin"); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var accessToken string
	if raw, err := auth.IssueAccessToken(h.Cfg, uid, []string{"admin"}); err == nil {
		accessToken = string(raw)
	}

	return response.JSON(c, 201, fiber.Map{
		"organization_id": orgID.String(),
		"role":              "admin",
		"access_token":      accessToken,
	})
}
