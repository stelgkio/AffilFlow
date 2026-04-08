package services

import (
	"context"
	"time"

	"github.com/stelgkio/affilflow/backend/internal/blockchain"
	"github.com/stelgkio/affilflow/backend/internal/repository"
	"github.com/stelgkio/affilflow/backend/pkg/retry"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/transfer"
)

// PayoutService batches approved commissions to Stripe Connect / PayPal.
type PayoutService struct {
	aff       *repository.AffiliateRepository
	bc        blockchain.Service
	stripeKey string
}

func NewPayoutService(aff *repository.AffiliateRepository, bc blockchain.Service, stripeSecret string) *PayoutService {
	if bc == nil {
		bc = blockchain.Noop{}
	}
	return &PayoutService{aff: aff, bc: bc, stripeKey: stripeSecret}
}

// Run pays all approved commissions (best-effort per row).
func (s *PayoutService) Run(ctx context.Context) error {
	if s.stripeKey != "" {
		stripe.Key = s.stripeKey
	}
	rows, err := s.aff.ListApprovedCommissions(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		extID := ""
		prov := "stripe"
		if row.StripeAcct != nil && *row.StripeAcct != "" && s.stripeKey != "" {
			params := &stripe.TransferParams{
				Amount:      stripe.Int64(row.AmountCents),
				Currency:    stripe.String("eur"),
				Destination: stripe.String(*row.StripeAcct),
			}
			tr, err := transfer.New(params)
			if err != nil {
				continue
			}
			extID = tr.ID
		} else if row.PayPalEmail != nil && *row.PayPalEmail != "" {
			prov = "paypal"
			extID = "pending-manual"
			// PayPal Payouts API requires batch setup; record for ops.
		} else {
			continue
		}
		if err := s.aff.MarkCommissionPaid(ctx, row.CommissionID); err != nil {
			continue
		}
		_ = s.aff.InsertPayoutRecord(ctx, row.AffiliateID, row.AmountCents, prov, extID, "completed")
		orderID := row.OrderID
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = retry.Do(ctx2, 3, time.Second, func() error {
				return s.bc.MarkPaid(ctx2, orderID)
			})
		}()
	}
	return nil
}
