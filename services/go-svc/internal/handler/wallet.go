package handler

import (
	"github.com/flotaro/be-test/services/go-svc/internal/middleware"
	"github.com/flotaro/be-test/services/go-svc/internal/request"
	"github.com/flotaro/be-test/services/go-svc/internal/response"
	"github.com/flotaro/be-test/services/go-svc/internal/service"
	"github.com/gofiber/fiber/v3"
)

type WalletHandler interface {
	Get(c fiber.Ctx) error
}

type walletHandler struct {
	service service.WalletService
	auth    middleware.AuthMiddleware
}

func NewWalletHandler(service service.WalletService, auth middleware.AuthMiddleware) WalletHandler {
	return &walletHandler{service: service, auth: auth}
}

func (h *walletHandler) Get(c fiber.Ctx) error {
	var request request.GetWalletRequest
	if err := c.Bind().URI(&request); err != nil {
		return invalidRequest(c, "invalid wallet path")
	}
	if request.UserID != h.auth.UserID(c) {
		return forbidden(c)
	}

	wallet, err := h.service.Get(c.Context(), request)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.NewErrorResponse("invalid_request", err.Error(), nil))
	}

	return c.JSON(wallet)
}
