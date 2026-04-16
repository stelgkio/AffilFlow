package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/config"
	"github.com/stelgkio/affilflow/backend/internal/models"
	"github.com/stelgkio/affilflow/backend/internal/randstr"
	"github.com/stelgkio/affilflow/backend/internal/repository"
)

// InviteService manages affiliate invites.
type InviteService struct {
	cfg        *config.Config
	invites    *repository.InviteRepository
	users      *repository.UserRepository
	affiliates *repository.AffiliateRepository
	campain    *repository.CampainRepository
	limits     *LimitService
}

func NewInviteService(cfg *config.Config, inv *repository.InviteRepository, u *repository.UserRepository, a *repository.AffiliateRepository, camp *repository.CampainRepository, l *LimitService) *InviteService {
	return &InviteService{cfg: cfg, invites: inv, users: u, affiliates: a, campain: camp, limits: l}
}

func hashInviteToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// Create returns plainToken for URL and inviteID.
func (s *InviteService) Create(ctx context.Context, campainID uuid.UUID, email *string, createdBy *string) (plainToken string, inviteID uuid.UUID, err error) {
	ok, _, max, err := s.limits.CanInviteAffiliate(ctx, campainID)
	if err != nil {
		return "", uuid.Nil, err
	}
	if !ok {
		return "", uuid.Nil, fmt.Errorf("invite limit reached (max %d)", max)
	}
	plainToken, err = randstr.Hex(24)
	if err != nil {
		return "", uuid.Nil, err
	}
	hash := hashInviteToken(plainToken)
	exp := time.Now().Add(14 * 24 * time.Hour)
	id, err := s.invites.Insert(ctx, campainID, email, hash, exp, createdBy)
	if err != nil {
		return "", uuid.Nil, err
	}
	return plainToken, id, nil
}

// JoinURL builds /join/{token} URL.
func (s *InviteService) JoinURL(plainToken string) string {
	base := strings.TrimRight(s.cfg.PublicAppBaseURL, "/")
	return fmt.Sprintf("%s/join/%s", base, plainToken)
}

// GetPendingInvite resolves a plain token to a pending invite (public validation).
func (s *InviteService) GetPendingInvite(ctx context.Context, plainToken string) (*models.AffiliateInvite, error) {
	hash := hashInviteToken(plainToken)
	return s.invites.GetPendingByHash(ctx, hash)
}

// AcceptInvite activates affiliate for user after authentication.
func (s *InviteService) Accept(ctx context.Context, plainToken, userID string, email *string) error {
	hash := hashInviteToken(plainToken)
	inv, err := s.invites.GetPendingByHash(ctx, hash)
	if err != nil || inv == nil {
		return fmt.Errorf("invalid or expired invite")
	}
	if err := s.users.Upsert(ctx, userID, email, &inv.CampainID); err != nil {
		return err
	}
	code, err := s.uniqueCode(ctx)
	if err != nil {
		return err
	}
	rate := 0.1
	if s.campain != nil {
		if co, err := s.campain.GetByID(ctx, inv.CampainID); err == nil && co != nil && co.DefaultCommissionRate > 0 && co.DefaultCommissionRate <= 1 {
			rate = co.DefaultCommissionRate
		}
	}
	if _, err := s.affiliates.Insert(ctx, inv.CampainID, userID, code, rate); err != nil {
		return err
	}
	return s.invites.MarkAccepted(ctx, inv.ID)
}

func (s *InviteService) uniqueCode(ctx context.Context) (string, error) {
	for i := 0; i < 20; i++ {
		c, err := randstr.Hex(4)
		if err != nil {
			return "", err
		}
		ok, err := s.affiliates.CodeExists(ctx, c)
		if err != nil {
			return "", err
		}
		if !ok {
			return c, nil
		}
	}
	return "", fmt.Errorf("could not generate code")
}

// CountPendingInvitesForCampain returns pending outbound invites for a program.
func (s *InviteService) CountPendingInvitesForCampain(ctx context.Context, campainID uuid.UUID) (int64, error) {
	return s.invites.CountPendingForCampain(ctx, campainID)
}
