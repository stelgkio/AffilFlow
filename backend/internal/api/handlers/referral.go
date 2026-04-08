package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/stelgkio/affilflow/backend/pkg/safeurl"
)

// ReferralRedirect records a click and redirects to the merchant landing or ?next=.
//
// @Summary Referral redirect
// @Description Records a click, sets attribution cookie, redirects (302)
// @Tags referrals
// @Param code path string true "Referral code"
// @Param next query string false "Allowed redirect URL (host must be allowlisted)"
// @Success 302 "Redirect to REDIRECT_BASE_URL or next"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /ref/{code} [get]
func (h *Handlers) ReferralRedirect(c *fiber.Ctx) error {
	code := strings.TrimSpace(c.Params("code"))
	if code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "code required")
	}

	ctx := c.UserContext()
	aff, err := h.affiliates.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if aff == nil {
		return fiber.NewError(fiber.StatusNotFound, "unknown referral code")
	}

	ua := c.Get("User-Agent")
	ipStr := c.IP()
	var ipPtr *string
	if ipStr != "" && ipStr != "0.0.0.0" {
		ipPtr = &ipStr
	}
	if err := h.referrals.InsertClick(ctx, aff.ID, code, ua, ipPtr); err != nil {
		return err
	}

	maxAge := int(h.Cfg.ReferralCookieTTL.Seconds())
	secure := h.Cfg.Env == "production"
	c.Cookie(&fiber.Cookie{
		Name:     h.Cfg.ReferralCookieName,
		Value:    code,
		Path:     "/",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: "Lax",
		MaxAge:   maxAge,
	})

	next := c.Query("next")
	target := safeurl.RedirectTarget(h.Cfg.RedirectAllowHosts, h.Cfg.RedirectBaseURL, next)
	return c.Redirect(target, fiber.StatusFound)
}
