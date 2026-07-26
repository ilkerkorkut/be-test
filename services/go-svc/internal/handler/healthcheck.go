package handler

import (
	"github.com/flotaro/be-test/services/go-svc/internal/enums"
	"github.com/flotaro/be-test/services/go-svc/internal/response"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type HealthCheckHandler interface {
	Liveness(c fiber.Ctx) error
}

type healthCheckHandler struct {
	logger *zap.Logger
}

func NewHealthCheckHandler(
	logger *zap.Logger,
) HealthCheckHandler {
	return &healthCheckHandler{
		logger: logger,
	}
}

// Liveness godoc
// @Tags Healthcheck
// @ID healthcheck.liveness
// @Description Liveness check
// @Produce  json
// @Success 200 {object} response.HealthResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /liveness [get]
func (h *healthCheckHandler) Liveness(c fiber.Ctx) error {
	return c.JSON(response.HealthResponse{Status: enums.HealthLivenessStatusOk})
}
