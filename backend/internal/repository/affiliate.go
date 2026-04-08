package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stelgkio/affilflow/backend/internal/models"
)

// AffiliateRepository loads affiliate rows.
type AffiliateRepository struct {
	pool *pgxpool.Pool
}

// NewAffiliateRepository constructs a repository.
func NewAffiliateRepository(pool *pgxpool.Pool) *AffiliateRepository {
	return &AffiliateRepository{pool: pool}
}

// GetByCode returns an affiliate by global referral code.
func (r *AffiliateRepository) GetByCode(ctx context.Context, code string) (*models.Affiliate, error) {
	const q = `
		SELECT id, organization_id, user_id, code, commission_rate::float8, status,
			stripe_connect_account_id, paypal_email, created_at, updated_at
		FROM affiliates
		WHERE code = $1 AND status = 'active'
	`
	var a models.Affiliate
	err := r.pool.QueryRow(ctx, q, code).Scan(
		&a.ID, &a.OrganizationID, &a.UserID, &a.Code, &a.CommissionRate, &a.Status,
		&a.StripeConnectAccountID, &a.PayPalEmail, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("affiliate by code: %w", err)
	}
	return &a, nil
}
