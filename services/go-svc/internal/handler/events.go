package handler

import (
	"github.com/flotaro/be-test/services/go-svc/internal/request"
	"github.com/flotaro/be-test/services/go-svc/internal/response"
	"github.com/flotaro/be-test/services/go-svc/internal/service"
	"github.com/gofiber/fiber/v3"
)

type EventsHandler interface {
	GetEvents(c fiber.Ctx) error
	GetOdds(c fiber.Ctx) error
}

type eventsHandler struct {
	service service.SportsService
}

func NewEventsHandler(service service.SportsService) EventsHandler {
	return &eventsHandler{service: service}
}

func (h *eventsHandler) GetEvents(c fiber.Ctx) error {
	var request request.GetEventsRequest
	if err := c.Bind().URI(&request); err != nil {
		return invalidRequest(c, "invalid event list path")
	}

	events, err := h.service.GetEvents(c.Context(), request)
	if err != nil {
		return serviceError(c, err)
	}

	return c.JSON(events)
}

func (h *eventsHandler) GetOdds(c fiber.Ctx) error {
	var request request.GetOddsRequest
	if err := c.Bind().URI(&request); err != nil {
		return invalidRequest(c, "invalid odds path")
	}
	if err := c.Bind().Query(&request); err != nil {
		return invalidRequest(c, "invalid odds query")
	}

	odds, err := h.service.GetOdds(c.Context(), request)
	if err != nil {
		return serviceError(c, err)
	}

	return c.JSON(odds)
}

func invalidRequest(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(response.NewErrorResponse("invalid_request", message, nil))
}
