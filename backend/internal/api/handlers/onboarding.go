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

// OnboardCompany POST /api/v1/onboarding/company — create merchant campain and promote user to merchant.
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
	if u.CampainID != nil {
		return fiber.NewError(fiber.StatusConflict, "user already belongs to a campain")
	}

	campainID, err := h.Campain.Create(ctx, name, &uid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if h.Sub != nil {
		if err := h.Sub.CreateFree(ctx, campainID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
	if err := h.User.SetCampainAndRole(ctx, uid, &campainID, "merchant"); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var accessToken string
	if raw, err := auth.IssueAccessToken(h.Cfg, uid, []string{"merchant"}); err == nil {
		accessToken = string(raw)
	}

	return response.JSON(c, 201, fiber.Map{
		"campain_id":      campainID.String(),
		"role":              "merchant",
		"access_token":      accessToken,
	})
}
