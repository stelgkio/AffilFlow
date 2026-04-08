package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stelgkio/affilflow/backend/internal/models"
)

// OrganizationRepository loads organizations.
type OrganizationRepository struct {
	pool *pgxpool.Pool
}

func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{pool: pool}
}

// Create inserts a new organization and returns its id.
func (r *OrganizationRepository) Create(ctx context.Context, name string) (uuid.UUID, error) {
	const q = `INSERT INTO organizations (name) VALUES ($1) RETURNING id`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, q, name).Scan(&id)
	return id, err
}

// CreateTx is like Create within an existing transaction.
func (r *OrganizationRepository) CreateTx(ctx context.Context, tx pgx.Tx, name string) (uuid.UUID, error) {
	const q = `INSERT INTO organizations (name) VALUES ($1) RETURNING id`
	var id uuid.UUID
	err := tx.QueryRow(ctx, q, name).Scan(&id)
	return id, err
}

// GetByID returns organization by id.
func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	const q = `
		SELECT id, name, slug, discovery_enabled, approval_mode, stripe_customer_id, created_at, updated_at
		FROM organizations WHERE id = $1
	`
	var o models.Organization
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&o.ID, &o.Name, &o.Slug, &o.DiscoveryEnabled, &o.ApprovalMode, &o.StripeCustomerID, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ListDiscoverable returns orgs with discovery_enabled.
func (r *OrganizationRepository) ListDiscoverable(ctx context.Context) ([]models.Organization, error) {
	const q = `
		SELECT id, name, slug, discovery_enabled, approval_mode, stripe_customer_id, created_at, updated_at
		FROM organizations WHERE discovery_enabled = true ORDER BY name
	`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Organization
	for rows.Next() {
		var o models.Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.DiscoveryEnabled, &o.ApprovalMode, &o.StripeCustomerID, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		return []models.Organization{}, nil
	}
	return out, nil
}

// UpdateStripeCustomer sets Stripe customer id on organization.
func (r *OrganizationRepository) UpdateStripeCustomer(ctx context.Context, orgID uuid.UUID, customerID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE organizations SET stripe_customer_id = $2, updated_at = now() WHERE id = $1`, orgID, customerID)
	return err
}

// GetByShopifyDomain resolves org from shop domain.
func (r *OrganizationRepository) GetByShopifyDomain(ctx context.Context, domain string) (*uuid.UUID, error) {
	const q = `SELECT organization_id FROM shopify_stores WHERE shop_domain = $1`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, q, domain).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// GetByWooCommerceURL resolves org from site base URL (exact match).
func (r *OrganizationRepository) GetByWooCommerceURL(ctx context.Context, baseURL string) (*uuid.UUID, error) {
	const q = `SELECT organization_id FROM woocommerce_stores WHERE site_base_url = $1`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, q, baseURL).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// RegisterShopifyStore links a shop domain to an org (admin/setup).
func (r *OrganizationRepository) RegisterShopifyStore(ctx context.Context, orgID uuid.UUID, domain string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO shopify_stores (organization_id, shop_domain) VALUES ($1, $2)
		ON CONFLICT (shop_domain) DO UPDATE SET organization_id = EXCLUDED.organization_id
	`, orgID, domain)
	return err
}

// RegisterWooCommerceStore links Woo base URL to org.
func (r *OrganizationRepository) RegisterWooCommerceStore(ctx context.Context, orgID uuid.UUID, baseURL string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO woocommerce_stores (organization_id, site_base_url) VALUES ($1, $2)
		ON CONFLICT (organization_id) DO UPDATE SET site_base_url = EXCLUDED.site_base_url
	`, orgID, baseURL)
	return err
}

// GetIDByStripeCustomer returns org id for Stripe customer id.
func (r *OrganizationRepository) GetIDByStripeCustomer(ctx context.Context, customerID string) (*uuid.UUID, error) {
	const q = `SELECT id FROM organizations WHERE stripe_customer_id = $1`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, q, customerID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// CountAffiliates returns active affiliates for org.
func (r *OrganizationRepository) CountAffiliates(ctx context.Context, orgID uuid.UUID) (int64, error) {
	const q = `SELECT COUNT(*) FROM affiliates WHERE organization_id = $1 AND status = 'active'`
	var n int64
	err := r.pool.QueryRow(ctx, q, orgID).Scan(&n)
	return n, err
}
