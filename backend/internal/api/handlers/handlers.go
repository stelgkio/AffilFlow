package handlers

import (
	"github.com/stelgkio/affilflow/backend/internal/repository"
)

// Handlers aggregates HTTP handlers.
type Handlers struct {
	*Deps
	affiliates *repository.AffiliateRepository
	referrals  *repository.ReferralRepository
}

// New constructs Handlers.
func New(d *Deps, aff *repository.AffiliateRepository, ref *repository.ReferralRepository) *Handlers {
	return &Handlers{Deps: d, affiliates: aff, referrals: ref}
}
