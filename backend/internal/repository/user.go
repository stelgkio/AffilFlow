package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stelgkio/affilflow/backend/internal/models"
)

// UserRepository persists users.
type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// GetByID returns a user by id.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	const q = `SELECT id, email, display_name, role, organization_id, password_hash, created_at, updated_at FROM users WHERE id = $1`
	var u models.User
	err := r.pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.OrganizationID, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Upsert inserts or updates user row (legacy / admin paths).
func (r *UserRepository) Upsert(ctx context.Context, id string, email *string, orgID *uuid.UUID) error {
	const q = `
		INSERT INTO users (id, email, organization_id, role, updated_at)
		VALUES ($1, $2, $3, 'affiliate', now())
		ON CONFLICT (id) DO UPDATE SET
			email = COALESCE(EXCLUDED.email, users.email),
			organization_id = COALESCE(EXCLUDED.organization_id, users.organization_id),
			updated_at = now()
	`
	_, err := r.pool.Exec(ctx, q, id, email, orgID)
	return err
}

// CreateOAuthUser inserts a new user with role and optional display name.
func (r *UserRepository) CreateOAuthUser(ctx context.Context, id string, email, displayName *string, role string) error {
	const q = `
		INSERT INTO users (id, email, display_name, role, updated_at)
		VALUES ($1, $2, $3, $4, now())
	`
	_, err := r.pool.Exec(ctx, q, id, email, displayName, role)
	return err
}

// UpsertOAuthUser creates or updates user from OAuth (by id).
func (r *UserRepository) UpsertOAuthUser(ctx context.Context, id string, email, displayName *string, role string) error {
	const q = `
		INSERT INTO users (id, email, display_name, role, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (id) DO UPDATE SET
			email = COALESCE(EXCLUDED.email, users.email),
			display_name = COALESCE(EXCLUDED.display_name, users.display_name),
			updated_at = now()
	`
	_, err := r.pool.Exec(ctx, q, id, email, displayName, role)
	return err
}

// SetOrganizationAndRole sets merchant org membership and role (e.g. after company onboarding).
func (r *UserRepository) SetOrganizationAndRole(ctx context.Context, userID string, orgID *uuid.UUID, role string) error {
	const q = `UPDATE users SET organization_id = $2, role = $3, updated_at = now() WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, userID, orgID, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// PatchEmailDisplayName updates email/display_name when pointers are non-nil.
func (r *UserRepository) PatchEmailDisplayName(ctx context.Context, id string, email, displayName *string) error {
	const q = `
		UPDATE users SET
			email = COALESCE($2, email),
			display_name = COALESCE($3, display_name),
			updated_at = now()
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, q, id, email, displayName)
	return err
}

// SetRole updates user role (e.g. promote to admin).
func (r *UserRepository) SetRole(ctx context.Context, userID, role string) error {
	const q = `UPDATE users SET role = $2, updated_at = now() WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, userID, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// GetByEmail finds first user with email (case-insensitive).
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	const q = `SELECT id, email, display_name, role, organization_id, password_hash, created_at, updated_at FROM users WHERE lower(email) = lower($1) LIMIT 1`
	var u models.User
	err := r.pool.QueryRow(ctx, q, email).Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.OrganizationID, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpsertAuthIdentity links provider to user.
func (r *UserRepository) UpsertAuthIdentity(ctx context.Context, provider, providerUserID, userID string, email *string, profile []byte) error {
	const q = `
		INSERT INTO auth_identities (provider, provider_user_id, user_id, email, profile, updated_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, now())
		ON CONFLICT (provider, provider_user_id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			email = COALESCE(EXCLUDED.email, auth_identities.email),
			profile = COALESCE(EXCLUDED.profile, auth_identities.profile),
			updated_at = now()
	`
	_, err := r.pool.Exec(ctx, q, provider, providerUserID, userID, email, profile)
	return err
}

// FindAuthIdentity returns user id for provider + provider user id.
func (r *UserRepository) FindAuthIdentity(ctx context.Context, provider, providerUserID string) (userID string, err error) {
	const q = `SELECT user_id FROM auth_identities WHERE provider = $1 AND provider_user_id = $2`
	err = r.pool.QueryRow(ctx, q, provider, providerUserID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

// InsertPasswordUserTx inserts a user row with a bcrypt password hash (and optional org) inside tx.
func (r *UserRepository) InsertPasswordUserTx(ctx context.Context, tx pgx.Tx, id string, email, displayName *string, passwordHash string, role string, orgID *uuid.UUID) error {
	const q = `
		INSERT INTO users (id, email, display_name, role, password_hash, organization_id, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
	`
	_, err := tx.Exec(ctx, q, id, email, displayName, role, passwordHash, orgID)
	return err
}
