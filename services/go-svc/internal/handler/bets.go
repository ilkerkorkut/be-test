package handler

import (
	"errors"

	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/middleware"
	"github.com/flotaro/be-test/services/go-svc/internal/request"
	"github.com/flotaro/be-test/services/go-svc/internal/response"
	"github.com/flotaro/be-test/services/go-svc/internal/service"
	"github.com/gofiber/fiber/v3"
)

type BetsHandler interface {
	Place(c fiber.Ctx) error
	Get(c fiber.Ctx) error
}

type betsHandler struct {
	service service.BetService
	auth    middleware.AuthMiddleware
}

func NewBetsHandler(service service.BetService, auth middleware.AuthMiddleware) BetsHandler {
	return &betsHandler{service: service, auth: auth}
}

func (h *betsHandler) Place(c fiber.Ctx) error {
	var request request.PlaceBetRequest
	if err := c.Bind().Body(&request); err != nil {
		return invalidRequest(c, "invalid request body")
	}
	if request.UserID != h.auth.UserID(c) {
		return forbidden(c)
	}

	bet, err := h.service.Place(c.Context(), request)
	if err != nil {
		return betError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(bet)
}

func (h *betsHandler) Get(c fiber.Ctx) error {
	var request request.GetBetsRequest
	if err := c.Bind().Query(&request); err != nil {
		return invalidRequest(c, "invalid bet history query")
	}
	if request.UserID != h.auth.UserID(c) {
		return forbidden(c)
	}
	if request.Limit == 0 {
		request.Limit = 30
	}

	bets, err := h.service.Get(c.Context(), request)
	if err != nil {
		return betError(c, err)
	}

	return c.JSON(bets)
}

func betError(c fiber.Ctx, err error) error {
	if errors.Is(err, database.ErrInsufficientBalance) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.NewErrorResponse("insufficient_balance", err.Error(), nil))
	}
	if errors.Is(err, database.ErrBetNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(response.NewErrorResponse("not_found", err.Error(), nil))
	}

	return c.Status(fiber.StatusBadRequest).JSON(response.NewErrorResponse("invalid_request", err.Error(), nil))
}

func forbidden(c fiber.Ctx) error {
	return c.Status(fiber.StatusForbidden).JSON(response.NewErrorResponse("forbidden", "token subject does not match requested user", nil))
}
