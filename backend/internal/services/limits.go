package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/repository"
)

// LimitService enforces subscription invite caps.
type LimitService struct {
	campain *repository.CampainRepository
	sub *repository.SubscriptionRepository
}

func NewLimitService(c *repository.CampainRepository, s *repository.SubscriptionRepository) *LimitService {
	return &LimitService{campain: c, sub: s}
}

// CanInviteAffiliate returns true if campain is under max_invites for current plan.
func (s *LimitService) CanInviteAffiliate(ctx context.Context, campainID uuid.UUID) (bool, int, int, error) {
	pk, err := s.sub.GetActivePlanKey(ctx, campainID)
	if err != nil {
		return false, 0, 0, err
	}
	max, err := s.sub.GetPlanLimit(ctx, pk)
	if err != nil {
		return false, 0, 0, err
	}
	n, err := s.campain.CountAffiliates(ctx, campainID)
	if err != nil {
		return false, 0, 0, err
	}
	if int(n) >= max {
		return false, int(n), max, nil
	}
	return true, int(n), max, nil
}
