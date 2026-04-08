package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/middleware"
	"github.com/stelgkio/affilflow/backend/pkg/response"
)

// CompanyDashboard GET /api/v1/dashboard/company (admin).
// Uses users.organization_id, or ?organization_id= for admins when the user row has no org yet (dev).
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

	var orgID *uuid.UUID
	if u != nil && u.OrganizationID != nil {
		orgID = u.OrganizationID
	}
	if orgID == nil {
		q := strings.TrimSpace(c.Query("organization_id"))
		if q != "" {
			id, err := uuid.Parse(q)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid organization_id")
			}
			orgID = &id
		}
	}
	if orgID == nil {
		return response.JSONError(c, fiber.StatusBadRequest, "no_organization",
			"link your user to an organization (users.organization_id) or pass ?organization_id= as an admin")
	}

	summary, leaders, err := h.Dash.CompanySummaryWithLeaders(ctx, *orgID, 10)
	if err != nil {
		return err
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"summary":  summary,
		"leaders":  leaders,
		"currency": "EUR",
		"note":     "Programs map to merchant organizations until a dedicated campaigns model exists.",
	})
}

// AffiliateDashboard GET /api/v1/dashboard/affiliate — earnings per program (organization) for this user.
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
