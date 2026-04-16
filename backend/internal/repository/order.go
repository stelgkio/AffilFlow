package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderRepository persists orders and commissions.
type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// UpsertOrder inserts or updates an order; returns order id.
func (r *OrderRepository) UpsertOrder(ctx context.Context, tx pgx.Tx, campainID uuid.UUID, externalID, source string, customerRef *string, totalCents int64, currency string, affiliateID *uuid.UUID, raw json.RawMessage) (orderID uuid.UUID, err error) {
	const q = `
		INSERT INTO orders (campain_id, external_id, source, customer_ref, total_cents, currency, affiliate_id, raw_payload, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		ON CONFLICT (campain_id, external_id, source) DO UPDATE SET
			total_cents = EXCLUDED.total_cents,
			currency = EXCLUDED.currency,
			affiliate_id = COALESCE(EXCLUDED.affiliate_id, orders.affiliate_id),
			raw_payload = EXCLUDED.raw_payload,
			updated_at = now()
		RETURNING id
	`
	err = tx.QueryRow(ctx, q, campainID, externalID, source, customerRef, totalCents, currency, affiliateID, raw).Scan(&orderID)
	return orderID, err
}

// CommissionExists checks if commission exists for order+affiliate.
func (r *OrderRepository) CommissionExists(ctx context.Context, tx pgx.Tx, orderID, affiliateID uuid.UUID) (bool, error) {
	var n int
	err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM commissions WHERE order_id = $1 AND affiliate_id = $2`, orderID, affiliateID).Scan(&n)
	return n > 0, err
}

// InsertCommission creates a commission row.
func (r *OrderRepository) InsertCommission(ctx context.Context, tx pgx.Tx, affiliateID, orderID uuid.UUID, amountCents int64, status string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO commissions (affiliate_id, order_id, amount_cents, status, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (order_id, affiliate_id) DO NOTHING
	`, affiliateID, orderID, amountCents, status)
	return err
}

// Pool returns underlying pool for Begin.
func (r *OrderRepository) Pool() *pgxpool.Pool { return r.pool }

// Begin starts a transaction.
func (r *OrderRepository) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

// GetAffiliateByID loads affiliate for rate lookup.
func (r *OrderRepository) GetAffiliateByID(ctx context.Context, id uuid.UUID) (rate float64, campainID uuid.UUID, err error) {
	const q = `SELECT commission_rate::float8, campain_id FROM affiliates WHERE id = $1 AND status = 'active'`
	err = r.pool.QueryRow(ctx, q, id).Scan(&rate, &campainID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, uuid.Nil, fmt.Errorf("affiliate not found")
	}
	return rate, campainID, err
}
