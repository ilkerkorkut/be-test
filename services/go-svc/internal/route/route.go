package route

import (
	"github.com/flotaro/be-test/services/go-svc/internal/handler"
	"github.com/flotaro/be-test/services/go-svc/internal/middleware"
	"github.com/gofiber/fiber/v3"
)

type AppContext struct {
	App *fiber.App
}

type Route interface {
	SetupRoutes(ac *AppContext)
}

type route struct {
	healthCheckHandler handler.HealthCheckHandler
	sportsHandler      handler.SportsHandler
	eventsHandler      handler.EventsHandler
	betsHandler        handler.BetsHandler
	walletHandler      handler.WalletHandler
	authMiddleware     middleware.AuthMiddleware
}

func NewRoute(
	healthCheckHandler handler.HealthCheckHandler,
	sportsHandler handler.SportsHandler,
	eventsHandler handler.EventsHandler,
	betsHandler handler.BetsHandler,
	walletHandler handler.WalletHandler,
	authMiddleware middleware.AuthMiddleware,
) Route {
	return &route{
		healthCheckHandler: healthCheckHandler,
		sportsHandler:      sportsHandler,
		eventsHandler:      eventsHandler,
		betsHandler:        betsHandler,
		walletHandler:      walletHandler,
		authMiddleware:     authMiddleware,
	}
}

func (r *route) SetupRoutes(ac *AppContext) {
	api := ac.App.Group("/")

	r.addHealthCheckRoutes(api)
	r.addProtectedRoutes(api)
}

func (r *route) addHealthCheckRoutes(fr fiber.Router) {
	fr.Get("/liveness", r.healthCheckHandler.Liveness)
	fr.Get("/health", r.healthCheckHandler.Liveness)
}

func (r *route) addProtectedRoutes(fr fiber.Router) {
	protected := fr.Group("", r.authMiddleware.RequireJWT)
	protected.Get("/sports", r.sportsHandler.GetSports)
	protected.Get("/sports/:key/events", r.eventsHandler.GetEvents)
	protected.Get("/events/:eventId/odds", r.eventsHandler.GetOdds)
	protected.Post("/bets", r.betsHandler.Place)
	protected.Get("/bets", r.betsHandler.Get)
	protected.Get("/wallet/:userId", r.walletHandler.Get)
}
