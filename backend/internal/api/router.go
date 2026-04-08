package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberSwagger "github.com/swaggo/fiber-swagger"

	"github.com/stelgkio/affilflow/backend/internal/api/handlers"
	"github.com/stelgkio/affilflow/backend/internal/config"
	"github.com/stelgkio/affilflow/backend/internal/middleware"
)

// NewFiberApp builds the Fiber application with routes.
func NewFiberApp(cfg *config.Config, h *handlers.Handlers) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
		BodyLimit:    4 * 1024 * 1024,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Get("/", h.Root)
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	app.Get("/health", h.Health)
	app.Get("/ref/:code", h.ReferralRedirect)

	app.Post("/webhooks/shopify/order-paid", h.ShopifyOrderPaid)
	app.Post("/webhooks/woocommerce/order-created", h.WooCommerceOrderCreated)
	app.Post("/webhooks/stripe/billing", h.StripeBillingWebhook)

	v1 := app.Group("/api/v1")
	v1.Get("/ping", h.Ping)
	v1.Get("/directory/programs", h.DirectoryPrograms)
	v1.Get("/invites/:token/validate", h.InviteValidate)

	v1.Get("/auth/providers/:provider/start", h.AuthOAuthStart)
	v1.Get("/auth/providers/:provider/callback", h.AuthOAuthCallback)
	v1.Post("/auth/register", h.AuthRegister)
	v1.Post("/auth/login", h.AuthLogin)

	protected := v1.Group("", middleware.AffilFlowJWT(cfg))
	protected.Post("/invites/:token/accept", h.InviteAccept)
	protected.Post("/directory/programs/:orgId/apply", h.DirectoryApply)
	protected.Post("/onboarding/company", h.OnboardCompany)
	protected.Post("/auth/logout", h.AuthLogout)
	protected.Get("/auth/me", h.AuthMe)
	protected.Get("/dashboard/affiliate", h.AffiliateDashboard)
	protected.Post("/organizations/:orgId/invites", middleware.RequireRoles("admin"), h.InviteCreate)
	protected.Post("/payouts/run", middleware.RequireRoles("admin"), h.PayoutRun)
	protected.Get("/dashboard/company", middleware.RequireRoles("admin"), h.CompanyDashboard)

	// Legacy path — prefer /auth/me
	protected.Get("/me", h.AuthMe)

	return app
}
