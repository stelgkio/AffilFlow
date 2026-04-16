// AffilFlow API — affiliate marketing platform backend.
// @title AffilFlow API
// @version 1.0
// @description REST API for the AffilFlow affiliate marketing platform (Fiber + PostgreSQL).
// @contact.name AffilFlow
// @host localhost:8080
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description AffilFlow access token: prefix Authorization with Bearer and a space.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stelgkio/affilflow/backend/internal/api"
	"github.com/stelgkio/affilflow/backend/internal/api/handlers"
	"github.com/stelgkio/affilflow/backend/internal/blockchain"
	"github.com/stelgkio/affilflow/backend/internal/config"
	"github.com/stelgkio/affilflow/backend/internal/database"
	"github.com/stelgkio/affilflow/backend/internal/repository"
	"github.com/stelgkio/affilflow/backend/internal/services"

	_ "github.com/stelgkio/affilflow/backend/apidocs" // swag generated OpenAPI
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if os.Getenv("AUTO_MIGRATE") == "true" || os.Getenv("AUTO_MIGRATE") == "1" {
		mp := os.Getenv("MIGRATIONS_PATH")
		if mp == "" {
			mp = "file://migrations"
		}
		if err := database.RunMigrations(cfg.DatabaseURL, mp); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		log.Println("migrations applied")
	}

	campainRepo := repository.NewCampainRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	subRepo := repository.NewSubscriptionRepository(pool)
	invRepo := repository.NewInviteRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	affRepo := repository.NewAffiliateRepository(pool)
	refRepo := repository.NewReferralRepository(pool)
	appRepo := repository.NewApplicationRepository(pool)
	dashRepo := repository.NewDashboardRepository(pool)

	bc := blockchain.Noop{}
	limitSvc := services.NewLimitService(campainRepo, subRepo)
	inviteSvc := services.NewInviteService(cfg, invRepo, userRepo, affRepo, campainRepo, limitSvc)
	billingSvc := services.NewBillingService(campainRepo, subRepo)
	orderSvc := services.NewOrderService(orderRepo, affRepo, bc)
	payoutSvc := services.NewPayoutService(affRepo, bc, cfg.StripeSecretKey)
	discoverySvc := services.NewDiscoveryService(campainRepo, appRepo, userRepo, affRepo, limitSvc)

	deps := &handlers.Deps{
		Cfg:       cfg,
		Pool:      pool,
		Order:     orderSvc,
		Invite:    inviteSvc,
		Limits:    limitSvc,
		Billing:   billingSvc,
		Payout:    payoutSvc,
		Discovery: discoverySvc,
		Campain:   campainRepo,
		User:      userRepo,
		Sub:       subRepo,
		AppRepo:   appRepo,
		Dash:      dashRepo,
	}
	h := handlers.New(deps, affRepo, refRepo)
	app := api.NewFiberApp(cfg, h)

	addr := ":" + cfg.HTTPPort
	go func() {
		log.Printf("listening on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
