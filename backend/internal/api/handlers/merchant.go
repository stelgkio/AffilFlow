package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/middleware"
	"github.com/stelgkio/affilflow/backend/internal/models"
	"github.com/stelgkio/affilflow/backend/internal/randstr"
	"github.com/stelgkio/affilflow/backend/internal/repository"
	"github.com/stelgkio/affilflow/backend/pkg/response"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func trimStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func optionalHTTPURL(s *string) (*string, error) {
	p := trimStringPtr(s)
	if p == nil {
		return nil, nil
	}
	low := strings.ToLower(*p)
	if !strings.HasPrefix(low, "http://") && !strings.HasPrefix(low, "https://") {
		return nil, fmt.Errorf("URL must start with http:// or https://")
	}
	return p, nil
}

func (h *Handlers) assertMerchantOwnsCampain(ctx context.Context, merchantUserID string, u *models.User, campainID uuid.UUID) error {
	camp, err := h.Campain.GetByID(ctx, campainID)
	if err != nil {
		return err
	}
	if camp == nil {
		return fiber.NewError(fiber.StatusNotFound, "campain not found")
	}
	if camp.OwnerUserID != nil && *camp.OwnerUserID == merchantUserID {
		return nil
	}
	if camp.OwnerUserID == nil && u.CampainID != nil && *u.CampainID == campainID {
		return nil
	}
	return fiber.NewError(fiber.StatusForbidden, "not your campain")
}

// merchantCampainID resolves the campain for this request: optional ?campain_id= UUID, else the user's default campain_id.
func (h *Handlers) merchantCampainID(c *fiber.Ctx) (*uuid.UUID, error) {
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	if uid == "" {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "missing user")
	}
	ctx := c.UserContext()
	u, err := h.User.GetByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}
	q := strings.TrimSpace(c.Query("campain_id"))
	if q != "" {
		id, err := uuid.Parse(q)
		if err != nil {
			return nil, response.JSONError(c, fiber.StatusBadRequest, "invalid_campain_id", "campain_id must be a UUID")
		}
		if err := h.assertMerchantOwnsCampain(ctx, uid, u, id); err != nil {
			return nil, err
		}
		return &id, nil
	}
	if u.CampainID == nil {
		return nil, response.JSONError(c, fiber.StatusBadRequest, "no_campain",
			"pass campain_id to work on a program, or set a default campain")
	}
	if err := h.assertMerchantOwnsCampain(ctx, uid, u, *u.CampainID); err != nil {
		return nil, err
	}
	return u.CampainID, nil
}

func (h *Handlers) genAffiliateCode(ctx context.Context) (string, error) {
	for i := 0; i < 20; i++ {
		c, err := randstr.Hex(4)
		if err != nil {
			return "", err
		}
		taken, err := h.affiliates.CodeExists(ctx, c)
		if err != nil {
			return "", err
		}
		if !taken {
			return c, nil
		}
	}
	return "", fmt.Errorf("could not generate referral code")
}

// MerchantProgramGet GET /api/v1/merchant/program — campain settings + store links + notification counts.
func (h *Handlers) MerchantProgramGet(c *fiber.Ctx) error {
	campainID, err := h.merchantCampainID(c)
	if err != nil {
		return err
	}
	ctx := c.UserContext()
	o, err := h.Campain.GetByID(ctx, *campainID)
	if err != nil || o == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "campain not found")
	}
	stores, err := h.Campain.GetLinkedStores(ctx, *campainID)
	if err != nil {
		return err
	}
	pendingApps, err := h.AppRepo.CountPendingByCampain(ctx, *campainID)
	if err != nil {
		return err
	}
	pendingInvites, err := h.Invite.CountPendingInvitesForCampain(ctx, *campainID)
	if err != nil {
		return err
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"campain":                    o,
		"linked_stores":              stores,
		"pending_applications_count": pendingApps,
		"pending_invites_count":      pendingInvites,
		"webhook_note":               "Each merchant registers their own Shopify domain and WordPress site URL here. WooCommerce signing secrets are generated per store; Shopify verification uses one platform SHOPIFY_WEBHOOK_SECRET plus X-Shopify-Shop-Domain routing.",
	})
}

type merchantProgramPatchBody struct {
	Name                    *string  `json:"name"`
	Slug                    *string  `json:"slug"`
	DiscoveryEnabled        *bool    `json:"discovery_enabled"`
	ApprovalMode            *string  `json:"approval_mode"`
	Tagline                 *string  `json:"tagline"`
	Description             *string  `json:"description"`
	BrandWebsiteURL         *string  `json:"brand_website_url"`
	TermsURL                *string  `json:"terms_url"`
	DefaultCommissionRate   *float64 `json:"default_commission_rate"`
	AttributionWindowDays   *int     `json:"attribution_window_days"`
}

// MerchantProgramPatch PATCH /api/v1/merchant/program — update partner program (campain) settings.
func (h *Handlers) MerchantProgramPatch(c *fiber.Ctx) error {
	campainID, err := h.merchantCampainID(c)
	if err != nil {
		return err
	}
	var body merchantProgramPatchBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}
	ctx := c.UserContext()
	o, err := h.Campain.GetByID(ctx, *campainID)
	if err != nil || o == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "campain not found")
	}
	name := o.Name
	if body.Name != nil {
		name = strings.TrimSpace(*body.Name)
		if name == "" {
			return response.JSONError(c, fiber.StatusBadRequest, "invalid_name", "name cannot be empty")
		}
	}
	slug := o.Slug
	if body.Slug != nil {
		s := strings.TrimSpace(strings.ToLower(*body.Slug))
		if s == "" {
			slug = nil
		} else {
			if !slugPattern.MatchString(s) || len(s) > 80 {
				return response.JSONError(c, fiber.StatusBadRequest, "invalid_slug",
					"use lowercase letters, numbers, and single hyphens (e.g. my-brand)")
			}
			taken, err := h.Campain.SlugTakenByOther(ctx, s, *campainID)
			if err != nil {
				return err
			}
			if taken {
				return response.JSONError(c, fiber.StatusConflict, "slug_taken", "that URL slug is already used")
			}
			slug = &s
		}
	}
	discovery := o.DiscoveryEnabled
	if body.DiscoveryEnabled != nil {
		discovery = *body.DiscoveryEnabled
	}
	mode := o.ApprovalMode
	if body.ApprovalMode != nil {
		m := strings.TrimSpace(strings.ToLower(*body.ApprovalMode))
		switch m {
		case "open", "request_to_join", "invite_only":
			mode = m
		default:
			return response.JSONError(c, fiber.StatusBadRequest, "invalid_mode",
				"approval_mode must be open, request_to_join, or invite_only")
		}
	}
	tagline := o.Tagline
	if body.Tagline != nil {
		tagline = trimStringPtr(body.Tagline)
	}
	description := o.Description
	if body.Description != nil {
		description = trimStringPtr(body.Description)
	}
	brandURL := o.BrandWebsiteURL
	if body.BrandWebsiteURL != nil {
		uu, err := optionalHTTPURL(body.BrandWebsiteURL)
		if err != nil {
			return response.JSONError(c, fiber.StatusBadRequest, "invalid_brand_url", err.Error())
		}
		brandURL = uu
	}
	termsURL := o.TermsURL
	if body.TermsURL != nil {
		uu, err := optionalHTTPURL(body.TermsURL)
		if err != nil {
			return response.JSONError(c, fiber.StatusBadRequest, "invalid_terms_url", err.Error())
		}
		termsURL = uu
	}
	defRate := o.DefaultCommissionRate
	if body.DefaultCommissionRate != nil {
		r := *body.DefaultCommissionRate
		if r <= 0 || r > 1 {
			return response.JSONError(c, fiber.StatusBadRequest, "invalid_commission",
				"default_commission_rate must be between 0 and 1 (e.g. 0.1 for 10%)")
		}
		defRate = r
	}
	attrDays := o.AttributionWindowDays
	if body.AttributionWindowDays != nil {
		d := *body.AttributionWindowDays
		if d < 1 || d > 365 {
			return response.JSONError(c, fiber.StatusBadRequest, "invalid_attribution_days",
				"attribution_window_days must be between 1 and 365")
		}
		attrDays = d
	}
	if err := h.Campain.UpdateCampainProgram(ctx, *campainID, repository.CampainProgramUpdate{
		Name: name, Slug: slug, DiscoveryEnabled: discovery, ApprovalMode: mode,
		Tagline: tagline, Description: description, BrandWebsiteURL: brandURL, TermsURL: termsURL,
		DefaultCommissionRate: defRate, AttributionWindowDays: attrDays,
	}); err != nil {
		return err
	}
	o2, err := h.Campain.GetByID(ctx, *campainID)
	if err != nil || o2 == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "campain not found")
	}
	return response.JSON(c, fiber.StatusOK, o2)
}

// MerchantApplicationsList GET /api/v1/merchant/applications
func (h *Handlers) MerchantApplicationsList(c *fiber.Ctx) error {
	campainID, err := h.merchantCampainID(c)
	if err != nil {
		return err
	}
	status := strings.TrimSpace(c.Query("status"))
	list, err := h.AppRepo.ListByCampain(c.UserContext(), *campainID, status)
	if err != nil {
		return err
	}
	return response.JSON(c, fiber.StatusOK, list)
}

// MerchantApplicationAccept POST /api/v1/merchant/applications/:applicationId/accept
func (h *Handlers) MerchantApplicationAccept(c *fiber.Ctx) error {
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing user")
	}
	appID, err := uuid.Parse(c.Params("applicationId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid application id")
	}
	ctx := c.UserContext()
	u, err := h.User.GetByID(ctx, uid)
	if err != nil || u == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}
	row, err := h.AppRepo.GetByID(ctx, appID)
	if err != nil {
		return err
	}
	if row == nil {
		return response.JSONError(c, fiber.StatusNotFound, "not_found", "application not found")
	}
	if err := h.assertMerchantOwnsCampain(ctx, uid, u, row.CampainID); err != nil {
		return err
	}
	campainID := row.CampainID
	if row.Status != "pending" {
		return response.JSONError(c, fiber.StatusConflict, "not_pending", "application is not pending")
	}
	if row.ApplicantUserID == nil || *row.ApplicantUserID == "" {
		return response.JSONError(c, fiber.StatusBadRequest, "no_applicant", "application has no applicant user id")
	}
	ok, _, max, err := h.Limits.CanInviteAffiliate(ctx, campainID)
	if err != nil {
		return err
	}
	if !ok {
		return response.JSONError(c, fiber.StatusConflict, "program_full",
			fmt.Sprintf("invite limit reached (max %d)", max))
	}
	existing, err := h.affiliates.GetByCampainAndUser(ctx, campainID, *row.ApplicantUserID)
	if err != nil {
		return err
	}
	if existing != nil {
		return response.JSONError(c, fiber.StatusConflict, "already_affiliate", "user is already an affiliate")
	}
	code, err := h.genAffiliateCode(ctx)
	if err != nil {
		return err
	}
	co, err := h.Campain.GetByID(ctx, campainID)
	if err != nil {
		return err
	}
	affRate := 0.1
	if co != nil && co.DefaultCommissionRate > 0 && co.DefaultCommissionRate <= 1 {
		affRate = co.DefaultCommissionRate
	}
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := h.User.UpsertTx(ctx, tx, *row.ApplicantUserID, row.ApplicantEmail, &campainID); err != nil {
		return err
	}
	if _, err := h.affiliates.InsertTx(ctx, tx, campainID, *row.ApplicantUserID, code, affRate); err != nil {
		return err
	}
	updated, err := h.AppRepo.SetStatusTx(ctx, tx, appID, campainID, "pending", "accepted")
	if err != nil {
		return err
	}
	if !updated {
		return response.JSONError(c, fiber.StatusConflict, "race", "application changed; retry")
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// MerchantApplicationReject POST /api/v1/merchant/applications/:applicationId/reject
func (h *Handlers) MerchantApplicationReject(c *fiber.Ctx) error {
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing user")
	}
	appID, err := uuid.Parse(c.Params("applicationId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid application id")
	}
	ctx := c.UserContext()
	u, err := h.User.GetByID(ctx, uid)
	if err != nil || u == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}
	row, err := h.AppRepo.GetByID(ctx, appID)
	if err != nil {
		return err
	}
	if row == nil {
		return response.JSONError(c, fiber.StatusNotFound, "not_found", "application not found")
	}
	if err := h.assertMerchantOwnsCampain(ctx, uid, u, row.CampainID); err != nil {
		return err
	}
	ok, err := h.AppRepo.SetStatus(ctx, appID, row.CampainID, "pending", "rejected")
	if err != nil {
		return err
	}
	if !ok {
		return response.JSONError(c, fiber.StatusNotFound, "not_found", "pending application not found")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// MerchantCampainsList GET /api/v1/merchant/campains — all programs owned by this merchant.
func (h *Handlers) MerchantCampainsList(c *fiber.Ctx) error {
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing user")
	}
	ctx := c.UserContext()
	list, err := h.Campain.ListForMerchantUser(ctx, uid)
	if err != nil {
		return err
	}
	u, err := h.User.GetByID(ctx, uid)
	if err != nil || u == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}
	var def *string
	if u.CampainID != nil {
		s := u.CampainID.String()
		def = &s
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"campains":            list,
		"default_campain_id": def,
	})
}

type merchantCampainCreateBody struct {
	Name                  string   `json:"name"`
	Tagline               *string  `json:"tagline"`
	Description           *string  `json:"description"`
	BrandWebsiteURL       *string  `json:"brand_website_url"`
	TermsURL              *string  `json:"terms_url"`
	DefaultCommissionRate *float64 `json:"default_commission_rate"`
	AttributionWindowDays *int     `json:"attribution_window_days"`
}

// MerchantCampainsCreate POST /api/v1/merchant/campains — create another program (campain).
func (h *Handlers) MerchantCampainsCreate(c *fiber.Ctx) error {
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing user")
	}
	var body merchantCampainCreateBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return response.JSONError(c, fiber.StatusBadRequest, "invalid_name", "name is required")
	}
	brandURL, err := optionalHTTPURL(body.BrandWebsiteURL)
	if err != nil {
		return response.JSONError(c, fiber.StatusBadRequest, "invalid_brand_url", err.Error())
	}
	termsURL, err := optionalHTTPURL(body.TermsURL)
	if err != nil {
		return response.JSONError(c, fiber.StatusBadRequest, "invalid_terms_url", err.Error())
	}
	defRate := 0.1
	if body.DefaultCommissionRate != nil {
		defRate = *body.DefaultCommissionRate
		if defRate <= 0 || defRate > 1 {
			return response.JSONError(c, fiber.StatusBadRequest, "invalid_commission",
				"default_commission_rate must be between 0 and 1 (e.g. 0.1 for 10%)")
		}
	}
	attrDays := 30
	if body.AttributionWindowDays != nil {
		attrDays = *body.AttributionWindowDays
		if attrDays < 1 || attrDays > 365 {
			return response.JSONError(c, fiber.StatusBadRequest, "invalid_attribution_days",
				"attribution_window_days must be between 1 and 365")
		}
	}
	ctx := c.UserContext()
	id, err := h.Campain.CreateCampain(ctx, repository.CampainCreateInput{
		Name: name, OwnerUserID: &uid,
		Tagline: trimStringPtr(body.Tagline), Description: trimStringPtr(body.Description),
		BrandWebsiteURL: brandURL, TermsURL: termsURL,
		DefaultCommissionRate: defRate, AttributionWindowDays: attrDays,
	})
	if err != nil {
		return err
	}
	if h.Sub != nil {
		if err := h.Sub.CreateFree(ctx, id); err != nil {
			return err
		}
	}
	u, err := h.User.GetByID(ctx, uid)
	if err != nil || u == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}
	if u.CampainID == nil {
		if err := h.User.SetCampainAndRole(ctx, uid, &id, "merchant"); err != nil {
			return err
		}
	}
	o, err := h.Campain.GetByID(ctx, id)
	if err != nil || o == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "campain not found")
	}
	return response.JSON(c, fiber.StatusCreated, o)
}

// MerchantCampainsSetDefault POST /api/v1/merchant/campains/:campainId/set-default — updates users.campain_id for API default.
func (h *Handlers) MerchantCampainsSetDefault(c *fiber.Ctx) error {
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing user")
	}
	id, err := uuid.Parse(c.Params("campainId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid campain id")
	}
	ctx := c.UserContext()
	u, err := h.User.GetByID(ctx, uid)
	if err != nil || u == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}
	if err := h.assertMerchantOwnsCampain(ctx, uid, u, id); err != nil {
		return err
	}
	if err := h.User.SetCampainAndRole(ctx, uid, &id, "merchant"); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// MerchantCampainsDelete DELETE /api/v1/merchant/campains/:campainId
func (h *Handlers) MerchantCampainsDelete(c *fiber.Ctx) error {
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing user")
	}
	id, err := uuid.Parse(c.Params("campainId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid campain id")
	}
	ctx := c.UserContext()
	u, err := h.User.GetByID(ctx, uid)
	if err != nil || u == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}
	if err := h.assertMerchantOwnsCampain(ctx, uid, u, id); err != nil {
		return err
	}
	wasDefault := u.CampainID != nil && *u.CampainID == id
	ok, err := h.Campain.DeleteOwnedByUser(ctx, id, uid)
	if err != nil {
		return err
	}
	if !ok {
		return response.JSONError(c, fiber.StatusNotFound, "not_found", "campain not found or not owned by you")
	}
	if wasDefault {
		list, err := h.Campain.ListForMerchantUser(ctx, uid)
		if err != nil {
			return err
		}
		var next *uuid.UUID
		if len(list) > 0 {
			next = &list[0].ID
		}
		if err := h.User.SetCampainAndRole(ctx, uid, next, "merchant"); err != nil {
			return err
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func normalizeShopDomain(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimSuffix(s, "/")
	return s
}

func normalizeWooBaseURL(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "/")
	return s
}

type shopifyLinkBody struct {
	ShopDomain string `json:"shop_domain"`
}

// MerchantIntegrationShopify POST /api/v1/merchant/integrations/shopify
func (h *Handlers) MerchantIntegrationShopify(c *fiber.Ctx) error {
	campainID, err := h.merchantCampainID(c)
	if err != nil {
		return err
	}
	var body shopifyLinkBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}
	domain := normalizeShopDomain(body.ShopDomain)
	if domain == "" || !strings.Contains(domain, ".") {
		return response.JSONError(c, fiber.StatusBadRequest, "invalid_domain", "enter your Shopify shop domain (e.g. your-store.myshopify.com)")
	}
	if err := h.Campain.RegisterShopifyStore(c.UserContext(), *campainID, domain); err != nil {
		return err
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"shop_domain": domain})
}

type wooLinkBody struct {
	SiteBaseURL string `json:"site_base_url"`
}

// MerchantIntegrationWooCommerce POST /api/v1/merchant/integrations/woocommerce
func (h *Handlers) MerchantIntegrationWooCommerce(c *fiber.Ctx) error {
	campainID, err := h.merchantCampainID(c)
	if err != nil {
		return err
	}
	var body wooLinkBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}
	base := normalizeWooBaseURL(body.SiteBaseURL)
	if base == "" || !strings.HasPrefix(base, "http") {
		return response.JSONError(c, fiber.StatusBadRequest, "invalid_url", "enter your full site base URL (e.g. https://shop.example.com)")
	}
	newSecret, err := h.Campain.UpsertWooCommerceStore(c.UserContext(), *campainID, base)
	if err != nil {
		return err
	}
	out := fiber.Map{
		"site_base_url": base,
		"copy_hint":     "If a signing secret is returned, paste it into WooCommerce → Settings → Advanced → Webhooks → Secret (shown only this once).",
	}
	if newSecret != nil {
		out["webhook_signing_secret"] = *newSecret
	}
	return response.JSON(c, fiber.StatusOK, out)
}

// MerchantRotateWooWebhookSecret POST /api/v1/merchant/integrations/woocommerce/rotate-secret
func (h *Handlers) MerchantRotateWooWebhookSecret(c *fiber.Ctx) error {
	campainID, err := h.merchantCampainID(c)
	if err != nil {
		return err
	}
	sec, err := h.Campain.RotateWooCommerceWebhookSecret(c.UserContext(), *campainID)
	if err != nil {
		return response.JSONError(c, fiber.StatusBadRequest, "no_store", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"webhook_signing_secret": sec,
		"copy_hint":              "Update the Secret field on the same webhook in WooCommerce, then send a test delivery.",
	})
}

// MerchantIntegrationsSetup GET /api/v1/merchant/integrations/setup — guided copy/paste for Shopify & WooCommerce.
func (h *Handlers) MerchantIntegrationsSetup(c *fiber.Ctx) error {
	campainID, err := h.merchantCampainID(c)
	if err != nil {
		return err
	}
	ctx := c.UserContext()
	stores, err := h.Campain.GetLinkedStores(ctx, *campainID)
	if err != nil {
		return err
	}
	base := strings.TrimRight(h.Cfg.AuthPublicBaseURL, "/")
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"api_base_url": base,
		"shopify": fiber.Map{
			"linked_shop_domain":     stores.ShopifyDomain,
			"webhook_delivery_url":   base + "/webhooks/shopify/order-paid",
			"webhook_event_suggestion": "Order payment",
			"steps": []string{
				"Enter your .myshopify.com domain below and click Save (one shop per program).",
				"In Shopify Admin → Settings → Notifications → Webhooks (or your Custom app), add a webhook: Event “Order payment”, Format JSON, URL = the delivery URL above.",
				"AffilFlow matches orders using the X-Shopify-Shop-Domain header. Your host must set SHOPIFY_WEBHOOK_SECRET to your Shopify app’s API client secret so signatures verify.",
			},
		},
		"woocommerce": fiber.Map{
			"linked_site_url":           stores.WooCommerceURL,
			"signing_secret_configured": stores.WooCommerceSigningSecretSet,
			"webhook_delivery_url":      base + "/webhooks/woocommerce/order-created",
			"webhook_topic_suggestion":  "Order created",
			"steps": []string{
				"Paste your live site URL (same as WordPress Settings → General → Site Address), then Save — AffilFlow generates a signing secret when needed.",
				"In WordPress: WooCommerce → Settings → Advanced → Webhooks → Add webhook. Name: AffilFlow, Status: Active, Topic: Order created, Delivery URL: copy from above, Secret: paste the signing secret from AffilFlow (use “Regenerate secret” if you lost it).",
				"WooCommerce sends header X-WC-Webhook-Source; AffilFlow uses it to route the order to your program. Optional legacy env WOOCOMMERCE_WEBHOOK_SECRET still verifies if set by the operator.",
			},
		},
	})
}

// MerchantWebhookURLs GET /api/v1/merchant/integrations/webhook-urls — copy/paste targets for store admin.
func (h *Handlers) MerchantWebhookURLs(c *fiber.Ctx) error {
	if _, err := h.merchantCampainID(c); err != nil {
		return err
	}
	base := strings.TrimRight(h.Cfg.AuthPublicBaseURL, "/")
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"shopify_order_paid_url":        base + "/webhooks/shopify/order-paid",
		"woocommerce_order_created_url": base + "/webhooks/woocommerce/order-created",
		"shopify_hmac_note":             "Operator sets SHOPIFY_WEBHOOK_SECRET once; each merchant registers their shop domain in the dashboard.",
	})
}
