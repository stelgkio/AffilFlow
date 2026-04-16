package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
func (r *ApplicationRepository) Insert(ctx context.Context, campainID uuid.UUID, userID *string, email *string) (uuid.UUID, error) {
	const q = `
		INSERT INTO affiliate_applications (campain_id, applicant_user_id, applicant_email, status, updated_at)
		VALUES ($1, $2, $3, 'pending', now())
		RETURNING id
	`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, q, campainID, userID, email).Scan(&id)
	return id, err
}

// AffiliateApplicationRow is a row for merchant review APIs.
type AffiliateApplicationRow struct {
	ID                uuid.UUID  `json:"id"`
	CampainID         uuid.UUID  `json:"campain_id"`
	ApplicantUserID   *string    `json:"applicant_user_id,omitempty"`
	ApplicantEmail    *string    `json:"applicant_email,omitempty"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
}

// ListByCampain returns applications for a campain, optionally filtered by status (e.g. "pending").
func (r *ApplicationRepository) ListByCampain(ctx context.Context, campainID uuid.UUID, status string) ([]AffiliateApplicationRow, error) {
	const q = `
		SELECT a.id, a.campain_id, a.applicant_user_id,
			COALESCE(a.applicant_email, u.email) AS applicant_email,
			a.status, a.created_at
		FROM affiliate_applications a
		LEFT JOIN users u ON u.id = a.applicant_user_id
		WHERE a.campain_id = $1
		  AND ($2 = '' OR a.status = $2)
		ORDER BY a.created_at DESC
	`
	rows, err := r.pool.Query(ctx, q, campainID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AffiliateApplicationRow
	for rows.Next() {
		var row AffiliateApplicationRow
		if err := rows.Scan(&row.ID, &row.CampainID, &row.ApplicantUserID, &row.ApplicantEmail, &row.Status, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		return []AffiliateApplicationRow{}, nil
	}
	return out, nil
}

// GetByID returns an application by id (any campain).
func (r *ApplicationRepository) GetByID(ctx context.Context, id uuid.UUID) (*AffiliateApplicationRow, error) {
	const q = `
		SELECT a.id, a.campain_id, a.applicant_user_id,
			COALESCE(a.applicant_email, u.email) AS applicant_email,
			a.status, a.created_at
		FROM affiliate_applications a
		LEFT JOIN users u ON u.id = a.applicant_user_id
		WHERE a.id = $1
	`
	var row AffiliateApplicationRow
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&row.ID, &row.CampainID, &row.ApplicantUserID, &row.ApplicantEmail, &row.Status, &row.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// CountPendingByCampain counts applications awaiting merchant action.
func (r *ApplicationRepository) CountPendingByCampain(ctx context.Context, campainID uuid.UUID) (int64, error) {
	const q = `SELECT COUNT(*) FROM affiliate_applications WHERE campain_id = $1 AND status = 'pending'`
	var n int64
	err := r.pool.QueryRow(ctx, q, campainID).Scan(&n)
	return n, err
}

// GetByIDAndCampain returns an application if it belongs to the campain.
func (r *ApplicationRepository) GetByIDAndCampain(ctx context.Context, id, campainID uuid.UUID) (*AffiliateApplicationRow, error) {
	const q = `
		SELECT a.id, a.campain_id, a.applicant_user_id,
			COALESCE(a.applicant_email, u.email) AS applicant_email,
			a.status, a.created_at
		FROM affiliate_applications a
		LEFT JOIN users u ON u.id = a.applicant_user_id
		WHERE a.id = $1 AND a.campain_id = $2
	`
	var row AffiliateApplicationRow
	err := r.pool.QueryRow(ctx, q, id, campainID).Scan(
		&row.ID, &row.CampainID, &row.ApplicantUserID, &row.ApplicantEmail, &row.Status, &row.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// SetStatus updates application status when currently in expectedFrom status.
func (r *ApplicationRepository) SetStatus(ctx context.Context, id, campainID uuid.UUID, expectedFrom, to string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE affiliate_applications
		SET status = $4, updated_at = now()
		WHERE id = $1 AND campain_id = $2 AND status = $3
	`, id, campainID, expectedFrom, to)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// SetStatusTx is SetStatus using an open transaction.
func (r *ApplicationRepository) SetStatusTx(ctx context.Context, tx pgx.Tx, id, campainID uuid.UUID, expectedFrom, to string) (bool, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE affiliate_applications
		SET status = $4, updated_at = now()
		WHERE id = $1 AND campain_id = $2 AND status = $3
	`, id, campainID, expectedFrom, to)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
