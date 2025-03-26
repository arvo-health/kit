package kit

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
)

// HealthCheckMiddleware sets up health check endpoints for liveness and readiness probes
// using the fiber-healthcheck middleware.
func HealthCheckMiddleware() fiber.Handler {
	livenessProbe := func(*fiber.Ctx) bool { return true }
	readinessProbe := func(*fiber.Ctx) bool { return true }

	return healthcheck.New(healthcheck.Config{
		LivenessEndpoint:  "/live",
		LivenessProbe:     livenessProbe,
		ReadinessEndpoint: "/ready",
		ReadinessProbe:    readinessProbe,
	})
}
