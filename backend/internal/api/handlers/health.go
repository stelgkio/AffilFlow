package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/stelgkio/affilflow/backend/pkg/response"
)

// Root lists service metadata and Swagger URL.
//
// @Summary Service root
// @Tags system
// @Produce json
// @Success 200 {object} IndexResponse
// @Router / [get]
func (h *Handlers) Root(c *fiber.Ctx) error {
	return response.JSON(c, 200, IndexResponse{
		Service: "affilflow",
		Version: "1.0",
		Docs:    "/swagger/index.html",
	})
}

// Health liveness probe.
//
// @Summary Liveness
// @Description Kubernetes-style health check
// @Tags system
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *Handlers) Health(c *fiber.Ctx) error {
	return response.JSON(c, 200, fiber.Map{"status": "ok"})
}

// Ping is a public API check.
//
// @Summary Ping API
// @Tags api
// @Produce json
// @Success 200 {object} PingResponse
// @Router /api/v1/ping [get]
func (h *Handlers) Ping(c *fiber.Ctx) error {
	return response.JSON(c, 200, fiber.Map{"message": "pong"})
}
