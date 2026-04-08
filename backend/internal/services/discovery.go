package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/randstr"
	"github.com/stelgkio/affilflow/backend/internal/repository"
)

// DiscoveryService handles public program directory and apply flows.
type DiscoveryService struct {
	org    *repository.OrganizationRepository
	app    *repository.ApplicationRepository
	users  *repository.UserRepository
	aff    *repository.AffiliateRepository
	limits *LimitService
}

func NewDiscoveryService(
	o *repository.OrganizationRepository,
	a *repository.ApplicationRepository,
	u *repository.UserRepository,
	aff *repository.AffiliateRepository,
	l *LimitService,
) *DiscoveryService {
	return &DiscoveryService{org: o, app: a, users: u, aff: aff, limits: l}
}

// Apply lets an authenticated user request to join a discoverable program.
func (s *DiscoveryService) Apply(ctx context.Context, orgID uuid.UUID, userID string, email *string) error {
	o, err := s.org.GetByID(ctx, orgID)
	if err != nil {
		return err
	}
	if o == nil {
		return fmt.Errorf("organization not found")
	}
	if !o.DiscoveryEnabled {
		return fmt.Errorf("program is not discoverable")
	}
	switch o.ApprovalMode {
	case "invite_only":
		return fmt.Errorf("this program is invite-only")
	case "request_to_join":
		_, err := s.app.Insert(ctx, orgID, &userID, email)
		return err
	case "open":
		ok, _, max, err := s.limits.CanInviteAffiliate(ctx, orgID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("program is at capacity (max %d affiliates)", max)
		}
		existing, err := s.aff.GetByOrgAndUser(ctx, orgID, userID)
		if err != nil {
			return err
		}
		if existing != nil {
			return fmt.Errorf("already an affiliate of this program")
		}
		if err := s.users.Upsert(ctx, userID, email, &orgID); err != nil {
			return err
		}
		code, err := s.uniqueAffiliateCode(ctx)
		if err != nil {
			return err
		}
		_, err = s.aff.Insert(ctx, orgID, userID, code, 0.1)
		return err
	default:
		return fmt.Errorf("unknown approval mode")
	}
}

func (s *DiscoveryService) uniqueAffiliateCode(ctx context.Context) (string, error) {
	for i := 0; i < 20; i++ {
		c, err := randstr.Hex(4)
		if err != nil {
			return "", err
		}
		ok, err := s.aff.CodeExists(ctx, c)
		if err != nil {
			return "", err
		}
		if !ok {
			return c, nil
		}
	}
	return "", fmt.Errorf("could not generate code")
}
