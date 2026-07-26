package handler

import (
	"github.com/flotaro/be-test/services/go-svc/internal/response"
	"github.com/flotaro/be-test/services/go-svc/internal/service"
	"github.com/gofiber/fiber/v3"
)

type SportsHandler interface {
	GetSports(c fiber.Ctx) error
}

type sportsHandler struct {
	service service.SportsService
}

func NewSportsHandler(service service.SportsService) SportsHandler {
	return &sportsHandler{service: service}
}

func (h *sportsHandler) GetSports(c fiber.Ctx) error {
	sports, err := h.service.GetSports(c.Context())
	if err != nil {
		return serviceError(c, err)
	}

	return c.JSON(sports)
}

func serviceError(c fiber.Ctx, err error) error {
	return c.Status(fiber.StatusBadGateway).JSON(response.NewErrorResponse("upstream_error", err.Error(), nil))
}
