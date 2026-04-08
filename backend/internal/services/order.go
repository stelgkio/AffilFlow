package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/blockchain"
	"github.com/stelgkio/affilflow/backend/internal/repository"
	"github.com/stelgkio/affilflow/backend/pkg/retry"
)

// OrderService processes external orders and commissions.
type OrderService struct {
	orders     *repository.OrderRepository
	affiliates *repository.AffiliateRepository
	bc         blockchain.Service
}

func NewOrderService(o *repository.OrderRepository, a *repository.AffiliateRepository, bc blockchain.Service) *OrderService {
	if bc == nil {
		bc = blockchain.Noop{}
	}
	return &OrderService{orders: o, affiliates: a, bc: bc}
}

// ProcessExternalOrder upserts order and creates commission when affiliate is set (idempotent).
func (s *OrderService) ProcessExternalOrder(ctx context.Context, orgID uuid.UUID, externalID, source string, customerRef *string, totalCents int64, currency string, affiliateID *uuid.UUID, raw json.RawMessage) error {
	tx, err := s.orders.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	orderID, err := s.orders.UpsertOrder(ctx, tx, orgID, externalID, source, customerRef, totalCents, currency, affiliateID, raw)
	if err != nil {
		return err
	}

	if affiliateID != nil {
		rate, affOrg, err := s.orders.GetAffiliateByID(ctx, *affiliateID)
		if err != nil {
			return err
		}
		if affOrg != orgID {
			return fmt.Errorf("affiliate does not belong to organization")
		}
		exists, err := s.orders.CommissionExists(ctx, tx, orderID, *affiliateID)
		if err != nil {
			return err
		}
		if exists {
			return tx.Commit(ctx)
		}
		amount := int64(math.Floor(float64(totalCents) * rate))
		if amount < 0 {
			amount = 0
		}
		if err := s.orders.InsertCommission(ctx, tx, *affiliateID, orderID, amount, "approved"); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		oid := orderID
		aid := *affiliateID
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = retry.Do(ctx2, 3, time.Second, func() error {
				return s.bc.RecordTransaction(ctx2, oid, aid, amount, "approved")
			})
		}()
		return nil
	}

	return tx.Commit(ctx)
}
