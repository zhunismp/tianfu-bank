package http

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/kruangsuriya/tianfu-bank/services/account-service/core/infrastructure/config"
)

type HttpServer struct {
	cfg      config.AppConfigProvider
	fiberApp *fiber.App
	router   fiber.Router
	basePath string
}

func NewHttpServer(cfg config.AppConfigProvider) *HttpServer {
	app := fiber.New(fiber.Config{
		AppName: cfg.GetServerName(),
	})

	app.Use(recover.New())
	app.Use(logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path}\n",
		TimeFormat: "2003/03/28 23:30:00",
		TimeZone:   "Asia/Bangkok",
	})
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Origin,X-PINGOTHER,Accept,Authorization,Content-Type,X-CSRF-Token",
		ExposeHeaders:    "Link",
		AllowCredentials: false,
		MaxAge:           300,
	}))

	apiGroup := app.Group(cfg.GetServerBaseApiPrefix())

	slog.Info("Fiber HTTP server initialized with middleware", "baseApiPrefix", cfg.GetServerBaseApiPrefix())

	return &HttpServer{
		cfg:      cfg,
		fiberApp: app,
		router:   apiGroup,
		basePath: cfg.GetServerBaseApiPrefix(),
	}
}

func (s *HttpServer) Start() {
	serverAddr := fmt.Sprintf("%s:%s", s.cfg.GetServerHost(), s.cfg.GetServerPort())
	slog.Info("Attempting to start HTTP server...", "serverAddress", serverAddr)

	go func() {
		if err := s.fiberApp.Listen(serverAddr); err != nil && err.Error() != "http: Server closed" {
			slog.Error("Failed to start HTTP server listener", "error", err)
		}
	}()

	slog.Info("HTTP server successfully started", "address", serverAddr)
}

func (s *HttpServer) Shutdown(ctx context.Context) {
	slog.Info("HTT server recived shutdown signal")

	if err := s.fiberApp.ShutdownWithContext(ctx); err != nil {
		slog.Error("Error during graceful shutdown HTTP server", "error", err)
		return
	}

	slog.Info("HTTP server shutdown gracefully")
}
