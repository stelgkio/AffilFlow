package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/stelgkio/affilflow/backend/internal/auth"
	"github.com/stelgkio/affilflow/backend/internal/config"
)

// AffilFlowJWT validates Bearer JWTs issued by this API (HS256, roles claim).
func AffilFlowJWT(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.JWTSkipValidation {
			c.Locals(LocalUserID, "dev-user")
			c.Locals(LocalRoles, []string{"merchant", "affiliate"})
			return c.Next()
		}

		if cfg.AuthJWTSecret == "" {
			return fiber.NewError(fiber.StatusInternalServerError, "AUTH_JWT_SECRET not configured")
		}

		authz := c.Get("Authorization")
		if authz == "" || !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
		}
		raw := strings.TrimSpace(authz[7:])

		sub, roles, err := auth.ParseAndValidateString(cfg, raw)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
		}

		c.Locals(LocalUserID, sub)
		c.Locals(LocalRoles, roles)

		return c.Next()
	}
}

// RequireRoles returns middleware that allows if any role matches.
func RequireRoles(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		v := c.Locals(LocalRoles)
		list, _ := v.([]string)
		for _, r := range list {
			if _, ok := allowed[r]; ok {
				return c.Next()
			}
		}
		return fiber.NewError(fiber.StatusForbidden, "insufficient role")
	}
}
