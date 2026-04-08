package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/repository"
)

// BillingService applies Stripe Billing webhook payloads to subscriptions.
type BillingService struct {
	org *repository.OrganizationRepository
	sub *repository.SubscriptionRepository
}

func NewBillingService(o *repository.OrganizationRepository, s *repository.SubscriptionRepository) *BillingService {
	return &BillingService{org: o, sub: s}
}

// ApplyStripeEvent parses a verified Stripe event JSON and updates DB.
func (s *BillingService) ApplyStripeEvent(ctx context.Context, raw []byte) error {
	var envelope struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	switch envelope.Type {
	case "customer.subscription.updated", "customer.subscription.created", "customer.subscription.deleted":
		var wrap struct {
			Object json.RawMessage `json:"object"`
		}
		if err := json.Unmarshal(envelope.Data, &wrap); err != nil {
			return err
		}
		var sub struct {
			ID               string `json:"id"`
			Customer         string `json:"customer"`
			Status           string `json:"status"`
			CurrentPeriodEnd int64  `json:"current_period_end"`
			Items            struct {
				Data []struct {
					Price struct {
						ID string `json:"id"`
					} `json:"price"`
				} `json:"data"`
			} `json:"items"`
		}
		if err := json.Unmarshal(wrap.Object, &sub); err != nil {
			return err
		}
		orgID, err := s.org.GetIDByStripeCustomer(ctx, sub.Customer)
		if err != nil {
			return err
		}
		if orgID == nil {
			return fmt.Errorf("unknown stripe customer %s", sub.Customer)
		}
		priceID := ""
		if len(sub.Items.Data) > 0 {
			priceID = sub.Items.Data[0].Price.ID
		}
		planKey, _ := s.sub.PlanKeyForStripePrice(ctx, priceID)
		var periodEnd *time.Time
		if sub.CurrentPeriodEnd > 0 {
			t := time.Unix(sub.CurrentPeriodEnd, 0).UTC()
			periodEnd = &t
		}
		st := sub.Status
		if st == "canceled" || st == "unpaid" {
			st = "canceled"
		}
		return s.sub.UpsertByStripe(ctx, *orgID, planKey, sub.ID, st, periodEnd)
	case "checkout.session.completed":
		var wrap struct {
			Object json.RawMessage `json:"object"`
		}
		if err := json.Unmarshal(envelope.Data, &wrap); err != nil {
			return err
		}
		var sess struct {
			Customer          string `json:"customer"`
			ClientReferenceID string `json:"client_reference_id"`
		}
		if err := json.Unmarshal(wrap.Object, &sess); err != nil {
			return err
		}
		if sess.ClientReferenceID != "" && sess.Customer != "" {
			id, err := uuid.Parse(sess.ClientReferenceID)
			if err == nil {
				return s.org.UpdateStripeCustomer(ctx, id, sess.Customer)
			}
		}
		return nil
	default:
		return nil
	}
}
