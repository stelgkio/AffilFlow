package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ApplicationRepository manages affiliate_applications.
type ApplicationRepository struct {
	pool *pgxpool.Pool
}

func NewApplicationRepository(pool *pgxpool.Pool) *ApplicationRepository {
	return &ApplicationRepository{pool: pool}
}

// Insert creates a pending application.
func (r *ApplicationRepository) Insert(ctx context.Context, orgID uuid.UUID, userID *string, email *string) (uuid.UUID, error) {
	const q = `
		INSERT INTO affiliate_applications (organization_id, applicant_user_id, applicant_email, status, updated_at)
		VALUES ($1, $2, $3, 'pending', now())
		RETURNING id
	`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, q, orgID, userID, email).Scan(&id)
	return id, err
}
