package handlers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/integration/shopify"
	"github.com/stelgkio/affilflow/backend/internal/integration/woocommerce"
	"github.com/stelgkio/affilflow/backend/internal/storeurl"
	"github.com/stelgkio/affilflow/backend/pkg/response"
	"github.com/stelgkio/affilflow/backend/pkg/retry"
)

// ShopifyOrderPaid POST /webhooks/shopify/order-paid
func (h *Handlers) ShopifyOrderPaid(c *fiber.Ctx) error {
	raw := c.Body()
	sig := c.Get("X-Shopify-Hmac-Sha256")
	verified := shopify.VerifyWebhook(h.Cfg.ShopifyWebhookSecret, raw, sig)
	if !verified && !(strings.EqualFold(h.Cfg.Env, "development") && h.Cfg.ShopifyWebhookSecret == "") {
		return fiber.ErrUnauthorized
	}
	domain := c.Get("X-Shopify-Shop-Domain")
	ctx := c.UserContext()
	var campainID *uuid.UUID
	if domain != "" {
		campainID, _ = h.Campain.GetByShopifyDomain(ctx, domain)
	}
	if campainID == nil && strings.EqualFold(h.Cfg.Env, "development") {
		campainID = h.Cfg.DefaultCampainUUID
	}
	if campainID == nil {
		return response.JSONError(c, fiber.StatusBadRequest, "unknown_shop",
			"register your Shopify shop domain under Merchant dashboard → Store integrations (no DEFAULT_CAMPAIN_ID fallback outside development)")
	}
	var payload struct {
		ID         json.RawMessage `json:"id"`
		TotalPrice string          `json:"total_price"`
		Currency   string          `json:"currency"`
		Customer   struct {
			ID json.RawMessage `json:"id"`
		} `json:"customer"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	extID := strings.Trim(string(payload.ID), `"`)
	total, err := strconv.ParseFloat(payload.TotalPrice, 64)
	if err != nil {
		return err
	}
	totalCents := int64(total * 100)
	cur := payload.Currency
	if cur == "" {
		cur = "EUR"
	}
	cust := strings.Trim(string(payload.Customer.ID), `"`)
	var custPtr *string
	if cust != "" {
		custPtr = &cust
	}
	rawCopy := append([]byte(nil), raw...)
	campain := *campainID
	go func() {
		bg := context.Background()
		_ = retry.Do(bg, 3, time.Second, func() error {
			return h.Order.ProcessExternalOrder(bg, campain, extID, "shopify", custPtr, totalCents, cur, nil, rawCopy)
		})
	}()
	return c.SendStatus(fiber.StatusOK)
}

// WooCommerceOrderCreated POST /webhooks/woocommerce/order-created
func (h *Handlers) WooCommerceOrderCreated(c *fiber.Ctx) error {
	raw := c.Body()
	sig := c.Get("X-WC-Webhook-Signature")
	ctx := c.UserContext()

	var campainID *uuid.UUID
	var storeSecret *string
	source := strings.TrimSpace(c.Get("X-WC-Webhook-Source"))
	if source != "" {
		cid, sec, err := h.Campain.ResolveWooCommerceWebhook(ctx, source)
		if err != nil {
			return err
		}
		campainID, storeSecret = cid, sec
	}
	if campainID == nil && strings.TrimSpace(h.Cfg.WooCommerceURL) != "" {
		legacy := storeurl.NormalizeWooSiteBase(h.Cfg.WooCommerceURL)
		cid, sec, err := h.Campain.ResolveWooCommerceWebhook(ctx, legacy)
		if err != nil {
			return err
		}
		campainID, storeSecret = cid, sec
	}

	verified := false
	if storeSecret != nil && *storeSecret != "" {
		verified = woocommerce.VerifySignature(*storeSecret, raw, sig)
	}
	if !verified && strings.TrimSpace(h.Cfg.WooCommerceWebhookSecret) != "" {
		verified = woocommerce.VerifySignature(h.Cfg.WooCommerceWebhookSecret, raw, sig)
	}
	if !verified {
		return fiber.ErrUnauthorized
	}

	var payload struct {
		ID       int64  `json:"id"`
		Total    string `json:"total"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if campainID == nil && strings.EqualFold(h.Cfg.Env, "development") {
		campainID = h.Cfg.DefaultCampainUUID
	}
	if campainID == nil {
		return response.JSONError(c, fiber.StatusBadRequest, "unknown_store",
			"save your WordPress site URL in Merchant dashboard → Store integrations so AffilFlow can match X-WC-Webhook-Source (legacy WOOCOMMERCE_URL is optional)")
	}
	extID := strconv.FormatInt(payload.ID, 10)
	total, err := strconv.ParseFloat(payload.Total, 64)
	if err != nil {
		total = 0
	}
	totalCents := int64(total * 100)
	cur := payload.Currency
	if cur == "" {
		cur = "EUR"
	}
	rawCopy := append([]byte(nil), raw...)
	campain := *campainID
	go func() {
		bg := context.Background()
		_ = retry.Do(bg, 3, time.Second, func() error {
			return h.Order.ProcessExternalOrder(bg, campain, extID, "woocommerce", nil, totalCents, cur, nil, rawCopy)
		})
	}()
	return c.SendStatus(fiber.StatusOK)
}
