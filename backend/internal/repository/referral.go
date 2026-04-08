package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReferralRepository records click events.
type ReferralRepository struct {
	pool *pgxpool.Pool
}

// NewReferralRepository constructs a repository.
func NewReferralRepository(pool *pgxpool.Pool) *ReferralRepository {
	return &ReferralRepository{pool: pool}
}

// InsertClick stores a referral click.
func (r *ReferralRepository) InsertClick(ctx context.Context, affiliateID uuid.UUID, code, userAgent string, ip *string) error {
	const q = `
		INSERT INTO referrals (affiliate_id, code, user_agent, ip)
		VALUES ($1, $2, $3, $4::inet)
	`
	var ipVal any
	if ip != nil {
		ipVal = *ip
	} else {
		ipVal = nil
	}
	_, err := r.pool.Exec(ctx, q, affiliateID, code, userAgent, ipVal)
	if err != nil {
		return fmt.Errorf("insert referral: %w", err)
	}
	return nil
}
