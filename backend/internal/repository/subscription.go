package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SubscriptionRepository manages SaaS subscriptions.
type SubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepository(pool *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool}
}

// CreateFree inserts a subscription row on the free plan for a new campain.
func (r *SubscriptionRepository) CreateFree(ctx context.Context, campainID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO subscriptions (campain_id, plan_key, status) VALUES ($1, 'free', 'active')
	`, campainID)
	return err
}

// CreateFreeTx is like CreateFree within an existing transaction.
func (r *SubscriptionRepository) CreateFreeTx(ctx context.Context, tx pgx.Tx, campainID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO subscriptions (campain_id, plan_key, status) VALUES ($1, 'free', 'active')
	`, campainID)
	return err
}

// UpsertByStripe replaces subscription row for a campain (single active SaaS sub).
func (r *SubscriptionRepository) UpsertByStripe(ctx context.Context, campainID uuid.UUID, planKey, stripeSubID, status string, periodEnd *time.Time) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM subscriptions WHERE campain_id = $1`, campainID)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO subscriptions (campain_id, plan_key, stripe_subscription_id, status, current_period_end, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
	`, campainID, planKey, stripeSubID, status, periodEnd)
	return err
}

// PlanKeyForStripePrice maps Stripe price id to plan_key.
func (r *SubscriptionRepository) PlanKeyForStripePrice(ctx context.Context, stripePriceID string) (string, error) {
	if stripePriceID == "" {
		return "free", nil
	}
	const q = `SELECT plan_key FROM subscription_plans WHERE stripe_price_id = $1`
	var pk string
	err := r.pool.QueryRow(ctx, q, stripePriceID).Scan(&pk)
	if errors.Is(err, pgx.ErrNoRows) {
		return "free", nil
	}
	if err != nil {
		return "free", err
	}
	return pk, nil
}

// GetPlanLimit returns max_invites for plan_key.
func (r *SubscriptionRepository) GetPlanLimit(ctx context.Context, planKey string) (int, error) {
	const q = `SELECT max_invites FROM subscription_plans WHERE plan_key = $1`
	var n int
	err := r.pool.QueryRow(ctx, q, planKey).Scan(&n)
	return n, err
}

// GetActivePlanKey returns plan_key for campain's active subscription or "free".
func (r *SubscriptionRepository) GetActivePlanKey(ctx context.Context, campainID uuid.UUID) (string, error) {
	const q = `
		SELECT plan_key FROM subscriptions
		WHERE campain_id = $1 AND status IN ('active', 'trialing', 'past_due')
		ORDER BY updated_at DESC LIMIT 1
	`
	var pk string
	err := r.pool.QueryRow(ctx, q, campainID).Scan(&pk)
	if errors.Is(err, pgx.ErrNoRows) {
		return "free", nil
	}
	if err != nil {
		return "free", err
	}
	return pk, nil
}
