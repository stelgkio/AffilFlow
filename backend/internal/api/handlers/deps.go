package handlers

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stelgkio/affilflow/backend/internal/config"
	"github.com/stelgkio/affilflow/backend/internal/repository"
	"github.com/stelgkio/affilflow/backend/internal/services"
)

// Deps aggregates injected collaborators for HTTP handlers.
type Deps struct {
	Cfg  *config.Config
	Pool *pgxpool.Pool

	Order     *services.OrderService
	Invite    *services.InviteService
	Limits    *services.LimitService
	Billing   *services.BillingService
	Payout    *services.PayoutService
	Discovery *services.DiscoveryService

	Campain *repository.CampainRepository
	User    *repository.UserRepository
	Sub     *repository.SubscriptionRepository
	AppRepo *repository.ApplicationRepository
	Dash    *repository.DashboardRepository
}
