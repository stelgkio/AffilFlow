package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stelgkio/affilflow/backend/internal/models"
	"github.com/stelgkio/affilflow/backend/internal/randstr"
	"github.com/stelgkio/affilflow/backend/internal/storeurl"
)

// CampainRepository loads campains.
type CampainRepository struct {
	pool *pgxpool.Pool
}

func NewCampainRepository(pool *pgxpool.Pool) *CampainRepository {
	return &CampainRepository{pool: pool}
}

const sqlCampainSelect = `id, name, slug, discovery_enabled, approval_mode,
	tagline, description, brand_website_url, terms_url,
	default_commission_rate, attribution_window_days,
	stripe_customer_id, owner_user_id, created_at, updated_at`

func nullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := strings.TrimSpace(ns.String)
	if s == "" {
		return nil
	}
	return &s
}

func scanCampainFromRow(scanner interface{ Scan(dest ...any) error }) (*models.Campain, error) {
	var c models.Campain
	var tag, desc, brand, terms, slug, stripe, owner sql.NullString
	err := scanner.Scan(
		&c.ID, &c.Name, &slug, &c.DiscoveryEnabled, &c.ApprovalMode,
		&tag, &desc, &brand, &terms,
		&c.DefaultCommissionRate, &c.AttributionWindowDays,
		&stripe, &owner,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.Slug = nullStringPtr(slug)
	c.Tagline = nullStringPtr(tag)
	c.Description = nullStringPtr(desc)
	c.BrandWebsiteURL = nullStringPtr(brand)
	c.TermsURL = nullStringPtr(terms)
	c.StripeCustomerID = nullStringPtr(stripe)
	if owner.Valid && owner.String != "" {
		s := owner.String
		c.OwnerUserID = &s
	}
	return &c, nil
}

// CampainCreateInput is used when inserting a new affiliate program (campain).
type CampainCreateInput struct {
	Name                  string
	OwnerUserID           *string
	Tagline               *string
	Description           *string
	BrandWebsiteURL       *string
	TermsURL              *string
	DefaultCommissionRate float64
	AttributionWindowDays int
}

func (in *CampainCreateInput) normalize() {
	if in.DefaultCommissionRate <= 0 || in.DefaultCommissionRate > 1 {
		in.DefaultCommissionRate = 0.1
	}
	if in.AttributionWindowDays < 1 || in.AttributionWindowDays > 365 {
		in.AttributionWindowDays = 30
	}
}

// CreateCampain inserts a full program row (use for merchant-driven creates).
func (r *CampainRepository) CreateCampain(ctx context.Context, in CampainCreateInput) (uuid.UUID, error) {
	in.normalize()
	owner := in.OwnerUserID
	if owner != nil && *owner == "" {
		owner = nil
	}
	const q = `
		INSERT INTO campains (
			name, owner_user_id, tagline, description, brand_website_url, terms_url,
			default_commission_rate, attribution_window_days
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, q,
		in.Name, owner, in.Tagline, in.Description, in.BrandWebsiteURL, in.TermsURL,
		in.DefaultCommissionRate, in.AttributionWindowDays,
	).Scan(&id)
	return id, err
}

// CreateCampainTx is like CreateCampain within a transaction.
func (r *CampainRepository) CreateCampainTx(ctx context.Context, tx pgx.Tx, in CampainCreateInput) (uuid.UUID, error) {
	in.normalize()
	owner := in.OwnerUserID
	if owner != nil && *owner == "" {
		owner = nil
	}
	const q = `
		INSERT INTO campains (
			name, owner_user_id, tagline, description, brand_website_url, terms_url,
			default_commission_rate, attribution_window_days
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	var id uuid.UUID
	err := tx.QueryRow(ctx, q,
		in.Name, owner, in.Tagline, in.Description, in.BrandWebsiteURL, in.TermsURL,
		in.DefaultCommissionRate, in.AttributionWindowDays,
	).Scan(&id)
	return id, err
}

// Create inserts a new campain and returns its id. ownerUserID is optional (merchants should set it).
func (r *CampainRepository) Create(ctx context.Context, name string, ownerUserID *string) (uuid.UUID, error) {
	return r.CreateCampain(ctx, CampainCreateInput{Name: name, OwnerUserID: ownerUserID})
}

// CreateTx is like Create within an existing transaction.
func (r *CampainRepository) CreateTx(ctx context.Context, tx pgx.Tx, name string, ownerUserID *string) (uuid.UUID, error) {
	return r.CreateCampainTx(ctx, tx, CampainCreateInput{Name: name, OwnerUserID: ownerUserID})
}

// GetByID returns campain by id.
func (r *CampainRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Campain, error) {
	q := `SELECT ` + sqlCampainSelect + ` FROM campains WHERE id = $1`
	c, err := scanCampainFromRow(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetDiscoverableByID returns a campain only if it exists and discovery is enabled.
func (r *CampainRepository) GetDiscoverableByID(ctx context.Context, id uuid.UUID) (*models.Campain, error) {
	q := `SELECT ` + sqlCampainSelect + ` FROM campains WHERE id = $1 AND discovery_enabled = true`
	c, err := scanCampainFromRow(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetDiscoverableBySlug returns a campain by slug only when discovery is enabled.
func (r *CampainRepository) GetDiscoverableBySlug(ctx context.Context, slug string) (*models.Campain, error) {
	q := `SELECT ` + sqlCampainSelect + ` FROM campains WHERE slug = $1 AND discovery_enabled = true`
	c, err := scanCampainFromRow(r.pool.QueryRow(ctx, q, slug))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ListDiscoverable returns campains with discovery_enabled.
func (r *CampainRepository) ListDiscoverable(ctx context.Context) ([]models.Campain, error) {
	q := `SELECT ` + sqlCampainSelect + ` FROM campains WHERE discovery_enabled = true ORDER BY name`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Campain
	for rows.Next() {
		c, err := scanCampainFromRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		return []models.Campain{}, nil
	}
	return out, nil
}

// ListForMerchantUser returns campains owned by this merchant user.
func (r *CampainRepository) ListForMerchantUser(ctx context.Context, ownerUserID string) ([]models.Campain, error) {
	q := `SELECT ` + sqlCampainSelect + ` FROM campains WHERE owner_user_id = $1 ORDER BY name`
	rows, err := r.pool.Query(ctx, q, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Campain
	for rows.Next() {
		c, err := scanCampainFromRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		return []models.Campain{}, nil
	}
	return out, nil
}

// DeleteOwnedByUser deletes a campain if it is owned by ownerUserID, or legacy-linked via users.campain_id with null owner.
func (r *CampainRepository) DeleteOwnedByUser(ctx context.Context, campainID uuid.UUID, ownerUserID string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM campains c
		WHERE c.id = $1
		  AND (
			c.owner_user_id = $2
			OR (
				c.owner_user_id IS NULL
				AND EXISTS (SELECT 1 FROM users u WHERE u.id = $2 AND u.campain_id = c.id)
			)
		  )
	`, campainID, ownerUserID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// UpdateStripeCustomer sets Stripe customer id on campain.
func (r *CampainRepository) UpdateStripeCustomer(ctx context.Context, campainID uuid.UUID, customerID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE campains SET stripe_customer_id = $2, updated_at = now() WHERE id = $1`, campainID, customerID)
	return err
}

// GetByShopifyDomain resolves campain from shop domain.
func (r *CampainRepository) GetByShopifyDomain(ctx context.Context, domain string) (*uuid.UUID, error) {
	const q = `SELECT campain_id FROM shopify_stores WHERE shop_domain = $1`
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

// GetByWooCommerceURL resolves campain from site base URL (exact match).
func (r *CampainRepository) GetByWooCommerceURL(ctx context.Context, baseURL string) (*uuid.UUID, error) {
	cid, _, err := r.ResolveWooCommerceWebhook(ctx, baseURL)
	return cid, err
}

// ResolveWooCommerceWebhook matches WooCommerce X-WC-Webhook-Source to a registered site_base_url.
func (r *CampainRepository) ResolveWooCommerceWebhook(ctx context.Context, webhookSource string) (*uuid.UUID, *string, error) {
	for _, candidate := range storeurl.WooSiteCandidates(webhookSource) {
		var id uuid.UUID
		var sec sql.NullString
		err := r.pool.QueryRow(ctx, `
			SELECT campain_id, webhook_secret FROM woocommerce_stores WHERE site_base_url = $1
		`, candidate).Scan(&id, &sec)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		var secretPtr *string
		if sec.Valid && sec.String != "" {
			s := sec.String
			secretPtr = &s
		}
		return &id, secretPtr, nil
	}
	return nil, nil, nil
}

// RegisterShopifyStore links a shop domain to a campain (merchant/setup).
func (r *CampainRepository) RegisterShopifyStore(ctx context.Context, campainID uuid.UUID, domain string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO shopify_stores (campain_id, shop_domain) VALUES ($1, $2)
		ON CONFLICT (shop_domain) DO UPDATE SET campain_id = EXCLUDED.campain_id
	`, campainID, domain)
	return err
}

// UpsertWooCommerceStore saves the merchant’s site URL. Generates a signing secret on first link (or if missing).
// When a new secret is created, it is returned once for the merchant to paste into WooCommerce → Webhooks → Secret.
func (r *CampainRepository) UpsertWooCommerceStore(ctx context.Context, campainID uuid.UUID, baseURL string) (*string, error) {
	base := storeurl.NormalizeWooSiteBase(baseURL)
	var ns sql.NullString
	err := r.pool.QueryRow(ctx, `SELECT webhook_secret FROM woocommerce_stores WHERE campain_id = $1`, campainID).Scan(&ns)
	if errors.Is(err, pgx.ErrNoRows) {
		sec, err := randstr.Hex(32)
		if err != nil {
			return nil, err
		}
		_, err = r.pool.Exec(ctx, `
			INSERT INTO woocommerce_stores (campain_id, site_base_url, webhook_secret)
			VALUES ($1, $2, $3)
		`, campainID, base, sec)
		if err != nil {
			return nil, err
		}
		return &sec, nil
	}
	if err != nil {
		return nil, err
	}
	_, err = r.pool.Exec(ctx, `UPDATE woocommerce_stores SET site_base_url = $2 WHERE campain_id = $1`, campainID, base)
	if err != nil {
		return nil, err
	}
	if !ns.Valid || ns.String == "" {
		sec, err := randstr.Hex(32)
		if err != nil {
			return nil, err
		}
		_, err = r.pool.Exec(ctx, `UPDATE woocommerce_stores SET webhook_secret = $2 WHERE campain_id = $1`, campainID, sec)
		if err != nil {
			return nil, err
		}
		return &sec, nil
	}
	return nil, nil
}

// RotateWooCommerceWebhookSecret replaces the WooCommerce webhook signing secret for this campain.
func (r *CampainRepository) RotateWooCommerceWebhookSecret(ctx context.Context, campainID uuid.UUID) (string, error) {
	sec, err := randstr.Hex(32)
	if err != nil {
		return "", err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE woocommerce_stores SET webhook_secret = $2 WHERE campain_id = $1`, campainID, sec)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", errors.New("save your WooCommerce site URL first")
	}
	return sec, nil
}

// GetIDByStripeCustomer returns campain id for Stripe customer id.
func (r *CampainRepository) GetIDByStripeCustomer(ctx context.Context, customerID string) (*uuid.UUID, error) {
	const q = `SELECT id FROM campains WHERE stripe_customer_id = $1`
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

// CountAffiliates returns active affiliates for campain.
func (r *CampainRepository) CountAffiliates(ctx context.Context, campainID uuid.UUID) (int64, error) {
	const q = `SELECT COUNT(*) FROM affiliates WHERE campain_id = $1 AND status = 'active'`
	var n int64
	err := r.pool.QueryRow(ctx, q, campainID).Scan(&n)
	return n, err
}

// SlugTakenByOther returns true if slug is used by a different campain.
func (r *CampainRepository) SlugTakenByOther(ctx context.Context, slug string, excludeCampainID uuid.UUID) (bool, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM campains WHERE slug = $1 AND id <> $2`, slug, excludeCampainID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CampainProgramUpdate is the full set of merchant-editable program fields.
type CampainProgramUpdate struct {
	Name                  string
	Slug                  *string
	DiscoveryEnabled      bool
	ApprovalMode          string
	Tagline               *string
	Description           *string
	BrandWebsiteURL       *string
	TermsURL              *string
	DefaultCommissionRate float64
	AttributionWindowDays int
}

// UpdateCampainProgram persists merchant program / campaign settings.
func (r *CampainRepository) UpdateCampainProgram(ctx context.Context, id uuid.UUID, p CampainProgramUpdate) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE campains SET
			name = $2, slug = $3, discovery_enabled = $4, approval_mode = $5,
			tagline = $6, description = $7, brand_website_url = $8, terms_url = $9,
			default_commission_rate = $10, attribution_window_days = $11,
			updated_at = now()
		WHERE id = $1
	`, id, p.Name, p.Slug, p.DiscoveryEnabled, p.ApprovalMode,
		p.Tagline, p.Description, p.BrandWebsiteURL, p.TermsURL,
		p.DefaultCommissionRate, p.AttributionWindowDays)
	return err
}

// LinkedStores holds store linkage for merchant setup UI.
type LinkedStores struct {
	ShopifyDomain               *string `json:"shopify_domain,omitempty"`
	WooCommerceURL              *string `json:"woocommerce_site_url,omitempty"`
	WooCommerceSigningSecretSet bool    `json:"woocommerce_signing_secret_set"`
}

// GetLinkedStores returns linked Shopify / WooCommerce rows for a campain.
func (r *CampainRepository) GetLinkedStores(ctx context.Context, campainID uuid.UUID) (LinkedStores, error) {
	var out LinkedStores
	var dom string
	err := r.pool.QueryRow(ctx, `SELECT shop_domain FROM shopify_stores WHERE campain_id = $1`, campainID).Scan(&dom)
	if err == nil {
		out.ShopifyDomain = &dom
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	var base string
	var hasSecret bool
	err = r.pool.QueryRow(ctx, `
		SELECT site_base_url,
			(webhook_secret IS NOT NULL AND length(trim(webhook_secret)) > 0)
		FROM woocommerce_stores WHERE campain_id = $1
	`, campainID).Scan(&base, &hasSecret)
	if err == nil {
		out.WooCommerceURL = &base
		out.WooCommerceSigningSecretSet = hasSecret
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	return out, nil
}
