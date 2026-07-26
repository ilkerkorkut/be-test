package main

import (
	"github.com/flotaro/be-test/services/go-svc/config"
	"github.com/flotaro/be-test/services/go-svc/internal/client"
	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/handler"
	"github.com/flotaro/be-test/services/go-svc/internal/middleware"
	"github.com/flotaro/be-test/services/go-svc/internal/route"
	"github.com/flotaro/be-test/services/go-svc/internal/service"
	"go.uber.org/zap"
)

func dependencyInjection(
	logger *zap.Logger,
	config config.ConfigurationManager,
	db database.Store,
) (route.Route, error) {
	healthcheckHandler := handler.NewHealthCheckHandler(
		logger,
	)
	authMiddleware := middleware.NewAuthMiddleware(config.GetAuthConfig().JWTSecret)
	apiClient := client.NewAPIClient(config, logger)
	sportsService, err := service.NewSportsService(apiClient, config.GetCacheConfig())
	if err != nil {
		return nil, err
	}
	walletService := service.NewWalletService(db, logger)
	betService := service.NewBetService(db, sportsService, logger)
	sportsHandler := handler.NewSportsHandler(sportsService)
	eventsHandler := handler.NewEventsHandler(sportsService)
	betsHandler := handler.NewBetsHandler(betService, authMiddleware)
	walletHandler := handler.NewWalletHandler(walletService, authMiddleware)

	r := route.NewRoute(
		healthcheckHandler,
		sportsHandler,
		eventsHandler,
		betsHandler,
		walletHandler,
		authMiddleware,
	)

	return r, nil
}
