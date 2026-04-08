package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/stelgkio/affilflow/backend/pkg/response"
	"github.com/stripe/stripe-go/v81/webhook"
)

// StripeBillingWebhook POST /webhooks/stripe/billing
func (h *Handlers) StripeBillingWebhook(c *fiber.Ctx) error {
	if h.Cfg.StripeBillingWebhookSecret == "" {
		return response.JSONError(c, fiber.StatusNotImplemented, "not_configured", "STRIPE_BILLING_WEBHOOK_SECRET not set")
	}
	payload := c.Body()
	sig := c.Get("Stripe-Signature")
	_, err := webhook.ConstructEventWithOptions(payload, sig, h.Cfg.StripeBillingWebhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return fiber.ErrUnauthorized
	}
	if err := h.Billing.ApplyStripeEvent(c.UserContext(), payload); err != nil {
		return response.JSONError(c, fiber.StatusBadRequest, "billing_apply_failed", err.Error())
	}
	return c.SendStatus(fiber.StatusOK)
}
