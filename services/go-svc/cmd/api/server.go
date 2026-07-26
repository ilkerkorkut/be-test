package main

import (
	"github.com/flotaro/be-test/services/go-svc/internal/middleware"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"go.uber.org/zap"

	"github.com/flotaro/be-test/services/go-svc/config"
	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/response"
	"github.com/flotaro/be-test/services/go-svc/internal/route"
)

type application struct {
	Logger   *zap.Logger
	Config   config.ConfigurationManager
	Database database.Store
}

func initApplication(a *application) (*fiber.App, error) {
	app := fiber.New(fiber.Config{
		// Override default error handler - Internal server err
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			a.Logger.Error("http request error", zap.String("method", c.Method()), zap.String("path", c.Path()), zap.Int("status", code), zap.Error(err))
			return c.Status(code).JSON(response.NewErrorResponse("internal", "internal server error", nil))
		},
	})

	a.addMiddlewares(app)

	r, err := dependencyInjection(
		a.Logger,
		a.Config,
		a.Database,
	)
	if err != nil {
		return nil, err
	}

	r.SetupRoutes(&route.AppContext{
		App: app,
	})

	// 404 handler
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(response.NewErrorResponse("not_found", "route not found", nil))
	})

	return app, nil
}

func (a *application) addMiddlewares(app *fiber.App) {
	app.Use(recover.New())
	app.Use(middleware.RequestLogger(a.Logger))
	app.Use(cors.New())
}
