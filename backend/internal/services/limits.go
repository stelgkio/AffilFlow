package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/repository"
)

// LimitService enforces subscription invite caps.
type LimitService struct {
	org *repository.OrganizationRepository
	sub *repository.SubscriptionRepository
}

func NewLimitService(o *repository.OrganizationRepository, s *repository.SubscriptionRepository) *LimitService {
	return &LimitService{org: o, sub: s}
}

// CanInviteAffiliate returns true if org is under max_invites for current plan.
func (s *LimitService) CanInviteAffiliate(ctx context.Context, orgID uuid.UUID) (bool, int, int, error) {
	pk, err := s.sub.GetActivePlanKey(ctx, orgID)
	if err != nil {
		return false, 0, 0, err
	}
	max, err := s.sub.GetPlanLimit(ctx, pk)
	if err != nil {
		return false, 0, 0, err
	}
	n, err := s.org.CountAffiliates(ctx, orgID)
	if err != nil {
		return false, 0, 0, err
	}
	if int(n) >= max {
		return false, int(n), max, nil
	}
	return true, int(n), max, nil
}
