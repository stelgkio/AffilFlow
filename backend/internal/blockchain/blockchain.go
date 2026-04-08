package blockchain

import (
	"context"

	"github.com/google/uuid"
)

// Service records commission lifecycle on Hyperledger Fabric (optional).
type Service interface {
	RecordTransaction(ctx context.Context, orderID, affiliateID uuid.UUID, commissionCents int64, status string) error
	MarkPaid(ctx context.Context, orderID uuid.UUID) error
}

// Noop implements Service without network calls (FABRIC_ENABLED=false).
type Noop struct{}

func (Noop) RecordTransaction(context.Context, uuid.UUID, uuid.UUID, int64, string) error {
	return nil
}

func (Noop) MarkPaid(context.Context, uuid.UUID) error {
	return nil
}
