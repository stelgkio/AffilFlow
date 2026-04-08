package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/stelgkio/affilflow/backend/pkg/response"
)

// PayoutRun POST /api/v1/payouts/run (admin) — batch approved commissions to Stripe Connect / PayPal.
//
// @Summary Run affiliate payouts (admin)
// @Tags payouts
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]bool
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payouts/run [post]
func (h *Handlers) PayoutRun(c *fiber.Ctx) error {
	if err := h.Payout.Run(c.UserContext()); err != nil {
		return response.JSONError(c, fiber.StatusInternalServerError, "payout_failed", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"ok": true})
}
