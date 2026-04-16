package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// Used only when APP_ENV=development and AUTH_JWT_SECRET is unset (local dev convenience).
const devJWTSecretFallback = "affilflow-dev-jwt-secret-change-me-min-32-chars!!"

// Config holds runtime configuration loaded from the environment.
type Config struct {
	Env         string
	HTTPPort    string
	DatabaseURL string

	// AffilFlow-issued JWT (replaces Keycloak for API auth).
	AuthJWTSecret   string
	AuthJWTIssuer   string
	AuthJWTAudience string
	// AuthPublicBaseURL is the external base URL of this API (e.g. http://localhost:8080) for OAuth redirect_uri.
	AuthPublicBaseURL string

	// OAuth (Google / Facebook) — register callback URLs as {AuthPublicBaseURL}/api/v1/auth/providers/{google|facebook}/callback
	OAuthGoogleClientID       string
	OAuthGoogleClientSecret   string
	OAuthFacebookClientID     string
	OAuthFacebookClientSecret string

	// AuthBootstrapAdminEmail if set, first OAuth sign-in with this email gets role merchant.
	AuthBootstrapAdminEmail string

	// JWTSkipValidation disables signature checks (local dev only; never in prod).
	JWTSkipValidation bool

	RedirectBaseURL    string
	ReferralCookieName string
	ReferralCookieTTL  time.Duration
	// RedirectAllowHosts comma-separated hostnames allowed for ?next= (e.g. localhost:3001,example.com)
	RedirectAllowHosts []string

	ShopifyWebhookSecret string

	WooCommerceURL            string
	WooCommerceConsumerKey    string
	WooCommerceConsumerSecret string
	WooCommerceWebhookSecret  string

	StripeSecretKey            string
	StripeWebhookSecret        string
	StripeBillingWebhookSecret string
	PayPalClientID             string
	PayPalSecret               string
	PayPalMode                 string // sandbox | live

	FabricEnabled       bool
	FabricNetworkConfig string
	FabricChannel       string
	FabricChaincode     string

	// PublicAppBaseURL is used to build invite links (e.g. https://app.example.com).
	PublicAppBaseURL string
	// DefaultCampainUUID optional campain for webhooks when store is not registered (dev only). Env: DEFAULT_CAMPAIN_ID
	DefaultCampainUUID *uuid.UUID
}

// Load reads .env if present and parses Config.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Env:               get("APP_ENV", "development"),
		HTTPPort:          get("HTTP_PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		AuthJWTSecret:     os.Getenv("AUTH_JWT_SECRET"),
		AuthJWTIssuer:     get("AUTH_JWT_ISSUER", "affilflow"),
		AuthJWTAudience:   get("AUTH_JWT_AUDIENCE", "affilflow-api"),
		AuthPublicBaseURL: get("AUTH_PUBLIC_BASE_URL", "http://localhost:8080"),

		OAuthGoogleClientID:       os.Getenv("OAUTH_GOOGLE_CLIENT_ID"),
		OAuthGoogleClientSecret:   os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET"),
		OAuthFacebookClientID:     os.Getenv("OAUTH_FACEBOOK_CLIENT_ID"),
		OAuthFacebookClientSecret: os.Getenv("OAUTH_FACEBOOK_CLIENT_SECRET"),

		AuthBootstrapAdminEmail: os.Getenv("AUTH_BOOTSTRAP_ADMIN_EMAIL"),

		RedirectBaseURL:            get("REDIRECT_BASE_URL", "https://example.com"),
		ReferralCookieName:         get("REFERRAL_COOKIE_NAME", "aff_ref"),
		StripeSecretKey:            os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret:        os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripeBillingWebhookSecret: os.Getenv("STRIPE_BILLING_WEBHOOK_SECRET"),
		PayPalClientID:             os.Getenv("PAYPAL_CLIENT_ID"),
		PayPalSecret:               os.Getenv("PAYPAL_SECRET"),
		PayPalMode:                 get("PAYPAL_MODE", "sandbox"),

		ShopifyWebhookSecret: os.Getenv("SHOPIFY_WEBHOOK_SECRET"),

		WooCommerceURL:            os.Getenv("WOOCOMMERCE_URL"),
		WooCommerceConsumerKey:    os.Getenv("WOOCOMMERCE_CONSUMER_KEY"),
		WooCommerceConsumerSecret: os.Getenv("WOOCOMMERCE_CONSUMER_SECRET"),
		WooCommerceWebhookSecret:  os.Getenv("WOOCOMMERCE_WEBHOOK_SECRET"),

		FabricNetworkConfig: os.Getenv("FABRIC_NETWORK_CONFIG"),
		FabricChannel:       os.Getenv("FABRIC_CHANNEL"),
		FabricChaincode:     os.Getenv("FABRIC_CHAINCODE"),

		PublicAppBaseURL: get("PUBLIC_APP_BASE_URL", "http://localhost:3001"),
	}

	if v := os.Getenv("JWT_SKIP_VALIDATION"); strings.EqualFold(v, "true") || v == "1" {
		cfg.JWTSkipValidation = true
	}
	if v := os.Getenv("FABRIC_ENABLED"); strings.EqualFold(v, "true") || v == "1" {
		cfg.FabricEnabled = true
	}

	ttlStr := get("REFERRAL_COOKIE_TTL_HOURS", "720")
	h, err := strconv.Atoi(ttlStr)
	if err != nil {
		return nil, fmt.Errorf("REFERRAL_COOKIE_TTL_HOURS: %w", err)
	}
	cfg.ReferralCookieTTL = time.Duration(h) * time.Hour

	if hosts := os.Getenv("REDIRECT_ALLOW_HOSTS"); hosts != "" {
		for _, h := range strings.Split(hosts, ",") {
			h = strings.TrimSpace(h)
			if h != "" {
				cfg.RedirectAllowHosts = append(cfg.RedirectAllowHosts, h)
			}
		}
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.AuthJWTSecret == "" {
		if strings.EqualFold(cfg.Env, "development") {
			cfg.AuthJWTSecret = devJWTSecretFallback
			log.Println("config: AUTH_JWT_SECRET not set; using development default. Set AUTH_JWT_SECRET in .env before production.")
		} else {
			return nil, fmt.Errorf("AUTH_JWT_SECRET is required when APP_ENV is not development")
		}
	}

	if v := os.Getenv("DEFAULT_CAMPAIN_ID"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("DEFAULT_CAMPAIN_ID: %w", err)
		}
		cfg.DefaultCampainUUID = &id
	}

	return cfg, nil
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
