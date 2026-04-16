package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stelgkio/affilflow/backend/internal/models"
)

// InviteRepository manages affiliate_invites.
type InviteRepository struct {
	pool *pgxpool.Pool
}

func NewInviteRepository(pool *pgxpool.Pool) *InviteRepository {
	return &InviteRepository{pool: pool}
}

// Insert creates a pending invite.
func (r *InviteRepository) Insert(ctx context.Context, campainID uuid.UUID, email *string, tokenHash string, expiresAt time.Time, createdBy *string) (uuid.UUID, error) {
	const q = `
		INSERT INTO affiliate_invites (campain_id, email, token_hash, expires_at, status, created_by_user_id)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id
	`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, q, campainID, email, tokenHash, expiresAt, createdBy).Scan(&id)
	return id, err
}

// GetPendingByHash returns invite if pending and not expired.
func (r *InviteRepository) GetPendingByHash(ctx context.Context, tokenHash string) (*models.AffiliateInvite, error) {
	const q = `
		SELECT id, campain_id, email, token_hash, expires_at, status, created_by_user_id, created_at
		FROM affiliate_invites
		WHERE token_hash = $1 AND status = 'pending' AND expires_at > now()
	`
	var m models.AffiliateInvite
	err := r.pool.QueryRow(ctx, q, tokenHash).Scan(
		&m.ID, &m.CampainID, &m.Email, &m.TokenHash, &m.ExpiresAt, &m.Status, &m.CreatedByUserID, &m.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// MarkAccepted sets invite status.
func (r *InviteRepository) MarkAccepted(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE affiliate_invites SET status = 'accepted' WHERE id = $1`, id)
	return err
}

// CountPendingForCampain counts invites you issued that are still open (partner not accepted yet).
func (r *InviteRepository) CountPendingForCampain(ctx context.Context, campainID uuid.UUID) (int64, error) {
	const q = `
		SELECT COUNT(*) FROM affiliate_invites
		WHERE campain_id = $1 AND status = 'pending' AND expires_at > now()
	`
	var n int64
	err := r.pool.QueryRow(ctx, q, campainID).Scan(&n)
	return n, err
}
