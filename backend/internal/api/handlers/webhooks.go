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
	"github.com/stelgkio/affilflow/backend/pkg/response"
	"github.com/stelgkio/affilflow/backend/pkg/retry"
)

// ShopifyOrderPaid POST /webhooks/shopify/order-paid
func (h *Handlers) ShopifyOrderPaid(c *fiber.Ctx) error {
	raw := c.Body()
	sig := c.Get("X-Shopify-Hmac-Sha256")
	if !shopify.VerifyWebhook(h.Cfg.ShopifyWebhookSecret, raw, sig) {
		return fiber.ErrUnauthorized
	}
	domain := c.Get("X-Shopify-Shop-Domain")
	ctx := c.UserContext()
	var orgID *uuid.UUID
	if domain != "" {
		orgID, _ = h.Org.GetByShopifyDomain(ctx, domain)
	}
	if orgID == nil {
		orgID = h.Cfg.DefaultOrgUUID
	}
	if orgID == nil {
		return response.JSONError(c, fiber.StatusBadRequest, "unknown_shop", "register shopify store or set DEFAULT_ORGANIZATION_ID")
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
	org := *orgID
	go func() {
		bg := context.Background()
		_ = retry.Do(bg, 3, time.Second, func() error {
			return h.Order.ProcessExternalOrder(bg, org, extID, "shopify", custPtr, totalCents, cur, nil, rawCopy)
		})
	}()
	return c.SendStatus(fiber.StatusOK)
}

// WooCommerceOrderCreated POST /webhooks/woocommerce/order-created
func (h *Handlers) WooCommerceOrderCreated(c *fiber.Ctx) error {
	raw := c.Body()
	if h.Cfg.WooCommerceWebhookSecret != "" {
		sig := c.Get("X-WC-Webhook-Signature")
		if sig == "" || sig != h.Cfg.WooCommerceWebhookSecret {
			return fiber.ErrUnauthorized
		}
	}
	var payload struct {
		ID       int64  `json:"id"`
		Total    string `json:"total"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	var orgID *uuid.UUID
	if h.Cfg.WooCommerceURL != "" {
		orgID, _ = h.Org.GetByWooCommerceURL(c.UserContext(), strings.TrimRight(h.Cfg.WooCommerceURL, "/"))
	}
	if orgID == nil {
		orgID = h.Cfg.DefaultOrgUUID
	}
	if orgID == nil {
		return response.JSONError(c, fiber.StatusBadRequest, "unknown_store", "set WOOCOMMERCE_URL store registration or DEFAULT_ORGANIZATION_ID")
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
	org := *orgID
	go func() {
		bg := context.Background()
		_ = retry.Do(bg, 3, time.Second, func() error {
			return h.Order.ProcessExternalOrder(bg, org, extID, "woocommerce", nil, totalCents, cur, nil, rawCopy)
		})
	}()
	return c.SendStatus(fiber.StatusOK)
}
