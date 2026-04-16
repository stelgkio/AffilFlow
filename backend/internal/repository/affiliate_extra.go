package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stelgkio/affilflow/backend/internal/models"
)

// InsertAffiliate creates an affiliate row.
func (r *AffiliateRepository) Insert(ctx context.Context, campainID uuid.UUID, userID, code string, rate float64) (uuid.UUID, error) {
	const q = `
		INSERT INTO affiliates (campain_id, user_id, code, commission_rate, status, updated_at)
		VALUES ($1, $2, $3, $4, 'active', now())
		RETURNING id
	`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, q, campainID, userID, code, rate).Scan(&id)
	return id, err
}

// InsertTx is like Insert using an open transaction.
func (r *AffiliateRepository) InsertTx(ctx context.Context, tx pgx.Tx, campainID uuid.UUID, userID, code string, rate float64) (uuid.UUID, error) {
	const q = `
		INSERT INTO affiliates (campain_id, user_id, code, commission_rate, status, updated_at)
		VALUES ($1, $2, $3, $4, 'active', now())
		RETURNING id
	`
	var id uuid.UUID
	err := tx.QueryRow(ctx, q, campainID, userID, code, rate).Scan(&id)
	return id, err
}

// CodeExists returns true if referral code is taken.
func (r *AffiliateRepository) CodeExists(ctx context.Context, code string) (bool, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM affiliates WHERE code = $1`, code).Scan(&n)
	return n > 0, err
}

// GetByCampainAndUser returns an affiliate row for campain + user, if any.
func (r *AffiliateRepository) GetByCampainAndUser(ctx context.Context, campainID uuid.UUID, userID string) (*models.Affiliate, error) {
	const q = `
		SELECT id, campain_id, user_id, code, commission_rate::float8, status,
			stripe_connect_account_id, paypal_email, created_at, updated_at
		FROM affiliates WHERE campain_id = $1 AND user_id = $2
	`
	var a models.Affiliate
	err := r.pool.QueryRow(ctx, q, campainID, userID).Scan(
		&a.ID, &a.CampainID, &a.UserID, &a.Code, &a.CommissionRate, &a.Status,
		&a.StripeConnectAccountID, &a.PayPalEmail, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// GetByID returns affiliate.
func (r *AffiliateRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Affiliate, error) {
	const q = `
		SELECT id, campain_id, user_id, code, commission_rate::float8, status,
			stripe_connect_account_id, paypal_email, created_at, updated_at
		FROM affiliates WHERE id = $1
	`
	var a models.Affiliate
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&a.ID, &a.CampainID, &a.UserID, &a.Code, &a.CommissionRate, &a.Status,
		&a.StripeConnectAccountID, &a.PayPalEmail, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListApprovedCommissions returns commissions ready for payout with affiliate payout fields.
type CommissionPayoutRow struct {
	CommissionID   uuid.UUID
	AffiliateID    uuid.UUID
	OrderID        uuid.UUID
	AmountCents    int64
	StripeAcct     *string
	PayPalEmail    *string
	CampainID uuid.UUID
}

func (r *AffiliateRepository) ListApprovedCommissions(ctx context.Context) ([]CommissionPayoutRow, error) {
	const q = `
		SELECT c.id, c.affiliate_id, c.order_id, c.amount_cents, a.stripe_connect_account_id, a.paypal_email, a.campain_id
		FROM commissions c
		JOIN affiliates a ON a.id = c.affiliate_id
		WHERE c.status = 'approved'
	`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommissionPayoutRow
	for rows.Next() {
		var row CommissionPayoutRow
		if err := rows.Scan(&row.CommissionID, &row.AffiliateID, &row.OrderID, &row.AmountCents, &row.StripeAcct, &row.PayPalEmail, &row.CampainID); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// MarkCommissionPaid updates commission status.
func (r *AffiliateRepository) MarkCommissionPaid(ctx context.Context, commissionID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE commissions SET status = 'paid', updated_at = now() WHERE id = $1`, commissionID)
	return err
}

// InsertPayoutRecord logs a payout batch line.
func (r *AffiliateRepository) InsertPayoutRecord(ctx context.Context, affID uuid.UUID, totalCents int64, provider, extID, status string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO payouts (affiliate_id, total_cents, provider, external_payout_id, status)
		VALUES ($1, $2, $3, $4, $5)
	`, affID, totalCents, provider, extID, status)
	return err
}
