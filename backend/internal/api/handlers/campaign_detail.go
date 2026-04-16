package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/models"
	"github.com/stelgkio/affilflow/backend/pkg/response"
)

// CampaignPublicCampain is safe to expose on the public campaign detail endpoint.
type CampaignPublicCampain struct {
	ID                      uuid.UUID `json:"id"`
	Name                    string    `json:"name"`
	Slug                    *string   `json:"slug,omitempty"`
	Tagline                 *string   `json:"tagline,omitempty"`
	Description             *string   `json:"description,omitempty"`
	BrandWebsiteURL         *string   `json:"brand_website_url,omitempty"`
	TermsURL                *string   `json:"terms_url,omitempty"`
	DefaultCommissionRate   float64   `json:"default_commission_rate"`
	AttributionWindowDays   int       `json:"attribution_window_days"`
	ApprovalMode            string    `json:"approval_mode"`
	DiscoveryEnabled        bool      `json:"discovery_enabled"`
}

// CampaignPublicPartner is a leaderboard row without internal user identifiers.
type CampaignPublicPartner struct {
	Code                   string `json:"code"`
	CommissionTotalCents   int64  `json:"commission_total_cents"`
	AttributedOrders       int64  `json:"attributed_orders"`
}

// CampaignDetailResponse is GET /api/v1/campaigns/:campaignRef
type CampaignDetailResponse struct {
	Campain     CampaignPublicCampain   `json:"campain"`
	Stats        CampaignPublicStats     `json:"stats"`
	TopPartners  []CampaignPublicPartner `json:"top_partners"`
}

// CampaignPublicStats aggregates program health for the public detail page.
type CampaignPublicStats struct {
	OrderCount               int64 `json:"order_count"`
	SalesTotalCents          int64 `json:"sales_total_cents"`
	CommissionsPendingCents  int64 `json:"commissions_pending_cents"`
	CommissionsApprovedCents int64 `json:"commissions_approved_cents"`
	CommissionsPaidCents     int64 `json:"commissions_paid_cents"`
	ActiveAffiliateCount     int64 `json:"active_affiliate_count"`
}

func campainToPublic(c *models.Campain) CampaignPublicCampain {
	return CampaignPublicCampain{
		ID:                    c.ID,
		Name:                  c.Name,
		Slug:                  c.Slug,
		Tagline:               c.Tagline,
		Description:           c.Description,
		BrandWebsiteURL:       c.BrandWebsiteURL,
		TermsURL:              c.TermsURL,
		DefaultCommissionRate: c.DefaultCommissionRate,
		AttributionWindowDays: c.AttributionWindowDays,
		ApprovalMode:          c.ApprovalMode,
		DiscoveryEnabled:      c.DiscoveryEnabled,
	}
}

// CampaignDetail GET /api/v1/campaigns/:campaignRef (UUID or discoverable slug)
func (h *Handlers) CampaignDetail(c *fiber.Ctx) error {
	ref := c.Params("campaignRef")
	if ref == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing campaign reference")
	}

	var campain *models.Campain
	var err error
	if id, parseErr := uuid.Parse(ref); parseErr == nil {
		campain, err = h.Campain.GetDiscoverableByID(c.UserContext(), id)
	} else {
		campain, err = h.Campain.GetDiscoverableBySlug(c.UserContext(), ref)
	}
	if err != nil {
		return err
	}
	if campain == nil {
		return fiber.NewError(fiber.StatusNotFound, "campaign not found")
	}

	summary, leaders, err := h.Dash.CompanySummaryWithLeaders(c.UserContext(), campain.ID, 5)
	if err != nil {
		return err
	}

	partners := make([]CampaignPublicPartner, 0, len(leaders))
	for _, row := range leaders {
		partners = append(partners, CampaignPublicPartner{
			Code:                 row.Code,
			CommissionTotalCents: row.CommissionSum,
			AttributedOrders:     row.OrderCount,
		})
	}

	out := CampaignDetailResponse{
		Campain: campainToPublic(campain),
		Stats: CampaignPublicStats{
			OrderCount:               summary.OrderCount,
			SalesTotalCents:          summary.SalesTotalCents,
			CommissionsPendingCents:  summary.CommissionsPending,
			CommissionsApprovedCents: summary.CommissionsApproved,
			CommissionsPaidCents:     summary.CommissionsPaid,
			ActiveAffiliateCount:     summary.ActiveAffiliateCount,
		},
		TopPartners: partners,
	}

	return response.JSON(c, fiber.StatusOK, out)
}
