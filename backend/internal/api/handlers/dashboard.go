package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/middleware"
	"github.com/stelgkio/affilflow/backend/pkg/response"
)

// CompanyDashboard GET /api/v1/dashboard/company (merchant).
// Uses ?campain_id= when present (must be owned by this merchant), otherwise users.campain_id.
func (h *Handlers) CompanyDashboard(c *fiber.Ctx) error {
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing user")
	}
	ctx := c.UserContext()

	u, err := h.User.GetByID(ctx, uid)
	if err != nil {
		return err
	}
	if u == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}

	var campainID *uuid.UUID
	q := strings.TrimSpace(c.Query("campain_id"))
	if q != "" {
		id, err := uuid.Parse(q)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid campain_id")
		}
		if strings.EqualFold(u.Role, "merchant") {
			if err := h.assertMerchantOwnsCampain(ctx, uid, u, id); err != nil {
				return err
			}
		}
		campainID = &id
	} else if u.CampainID != nil {
		campainID = u.CampainID
	}
	if campainID == nil {
		return response.JSONError(c, fiber.StatusBadRequest, "no_campain",
			"link your user to a campain or pass ?campain_id=")
	}

	summary, leaders, err := h.Dash.CompanySummaryWithLeaders(ctx, *campainID, 10)
	if err != nil {
		return err
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"summary":  summary,
		"leaders":  leaders,
		"currency": "EUR",
		"note":     "Programs map to merchant campains until a dedicated campaigns model exists.",
	})
}

// AffiliateDashboard GET /api/v1/dashboard/affiliate — earnings per program (campain) for this user.
func (h *Handlers) AffiliateDashboard(c *fiber.Ctx) error {
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing user")
	}
	rows, err := h.Dash.AffiliateProgramStats(c.UserContext(), uid)
	if err != nil {
		return err
	}
	var accruedTotal, paidTotal int64
	for _, r := range rows {
		accruedTotal += r.AccruedCents
		paidTotal += r.PaidCents
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"programs": rows,
		"totals": fiber.Map{
			"accrued_cents": accruedTotal,
			"paid_cents":    paidTotal,
		},
		"currency":    "EUR",
		"payout_note": "Pending + approved amounts are owed until your payout run; schedule is set by each merchant program.",
	})
}
