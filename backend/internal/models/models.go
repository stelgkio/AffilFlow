package models

import (
	"time"

	"github.com/google/uuid"
)

// Affiliate is a row in affiliates.
type Affiliate struct {
	ID                     uuid.UUID `db:"id"`
	CampainID              uuid.UUID `db:"campain_id"`
	UserID                 string    `db:"user_id"`
	Code                   string    `db:"code"`
	CommissionRate         float64   `db:"commission_rate"`
	Status                 string    `db:"status"`
	StripeConnectAccountID *string   `db:"stripe_connect_account_id"`
	PayPalEmail            *string   `db:"paypal_email"`
	CreatedAt              time.Time `db:"created_at"`
	UpdatedAt              time.Time `db:"updated_at"`
}

// Campain row.
type Campain struct {
	ID                      uuid.UUID `db:"id" json:"id"`
	Name                    string    `db:"name" json:"name"`
	Slug                    *string   `db:"slug" json:"slug,omitempty"`
	DiscoveryEnabled        bool      `db:"discovery_enabled" json:"discovery_enabled"`
	ApprovalMode            string    `db:"approval_mode" json:"approval_mode"`
	Tagline                 *string   `db:"tagline" json:"tagline,omitempty"`
	Description           *string   `db:"description" json:"description,omitempty"`
	BrandWebsiteURL         *string   `db:"brand_website_url" json:"brand_website_url,omitempty"`
	TermsURL                *string   `db:"terms_url" json:"terms_url,omitempty"`
	DefaultCommissionRate   float64   `db:"default_commission_rate" json:"default_commission_rate"`
	AttributionWindowDays   int       `db:"attribution_window_days" json:"attribution_window_days"`
	StripeCustomerID        *string   `db:"stripe_customer_id" json:"stripe_customer_id,omitempty"`
	OwnerUserID             *string   `db:"owner_user_id" json:"owner_user_id,omitempty"`
	CreatedAt               time.Time `db:"created_at" json:"created_at"`
	UpdatedAt               time.Time `db:"updated_at" json:"updated_at"`
}

// User is the app account (OAuth or legacy external IdP subject in id).
type User struct {
	ID             string     `db:"id"`
	Email          *string    `db:"email"`
	DisplayName    *string    `db:"display_name"`
	Role           string     `db:"role"`
	CampainID      *uuid.UUID `db:"campain_id"`
	PasswordHash   *string    `db:"password_hash" json:"-"` // set only for email/password auth
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

// AuthIdentity links an OAuth provider account to a user.
type AuthIdentity struct {
	ID             uuid.UUID `db:"id"`
	Provider       string    `db:"provider"`
	ProviderUserID string    `db:"provider_user_id"`
	UserID         string    `db:"user_id"`
	Email          *string   `db:"email"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// Subscription row.
type Subscription struct {
	ID                   uuid.UUID  `db:"id"`
	CampainID            uuid.UUID  `db:"campain_id"`
	PlanKey              string     `db:"plan_key"`
	StripeSubscriptionID *string    `db:"stripe_subscription_id"`
	Status               string     `db:"status"`
	CurrentPeriodEnd     *time.Time `db:"current_period_end"`
	CreatedAt            time.Time  `db:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at"`
}

// SubscriptionPlan catalog row.
type SubscriptionPlan struct {
	PlanKey       string  `db:"plan_key"`
	PriceEurCents int     `db:"price_eur_cents"`
	MaxInvites    int     `db:"max_invites"`
	StripePriceID *string `db:"stripe_price_id"`
}

// AffiliateInvite pending acceptance.
type AffiliateInvite struct {
	ID              uuid.UUID `db:"id"`
	CampainID       uuid.UUID `db:"campain_id"`
	Email           *string   `db:"email"`
	TokenHash       string    `db:"token_hash"`
	ExpiresAt       time.Time `db:"expires_at"`
	Status          string    `db:"status"`
	CreatedByUserID *string   `db:"created_by_user_id"`
	CreatedAt       time.Time `db:"created_at"`
}

// Order normalized from external systems.
type Order struct {
	ID             uuid.UUID  `db:"id"`
	CampainID      uuid.UUID  `db:"campain_id"`
	ExternalID     string     `db:"external_id"`
	Source         string     `db:"source"`
	CustomerRef    *string    `db:"customer_ref"`
	TotalCents     int64      `db:"total_cents"`
	Currency       string     `db:"currency"`
	ReferralID     *uuid.UUID `db:"referral_id"`
	AffiliateID    *uuid.UUID `db:"affiliate_id"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

// Commission owed to an affiliate for an order.
type Commission struct {
	ID          uuid.UUID `db:"id"`
	AffiliateID uuid.UUID `db:"affiliate_id"`
	OrderID     uuid.UUID `db:"order_id"`
	AmountCents int64     `db:"amount_cents"`
	Status      string    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// AffiliateApplication for directory apply flow.
type AffiliateApplication struct {
	ID              uuid.UUID `db:"id"`
	CampainID       uuid.UUID `db:"campain_id"`
	ApplicantUserID *string   `db:"applicant_user_id"`
	ApplicantEmail  *string   `db:"applicant_email"`
	Status          string    `db:"status"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}
