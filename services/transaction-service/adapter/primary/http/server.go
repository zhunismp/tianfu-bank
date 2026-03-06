package http

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/zhunismp/tianfu-bank/services/transaction-service/core/infrastructure/config"
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
	app.Use(logger.New(logger.Config{
		Format:       "${time} ${status} - ${method} ${path}\n",
		TimeFormat:   "2006/01/02 15:04:05",
		TimeInterval: 0,
		TimeZone:     "Asia/Bangkok",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Origin,X-PINGOTHER,Accept,Authorization,Content-Type,X-CSRF-Token,X-Idempotency-Key",
		ExposeHeaders:    "Link",
		AllowCredentials: false,
		MaxAge:           300,
	}))

	apiGroup := app.Group(cfg.GetServerBaseApiPrefix())

	slog.Info("Fiber HTTP server initialized", "baseApiPrefix", cfg.GetServerBaseApiPrefix())

	return &HttpServer{
		cfg:      cfg,
		fiberApp: app,
		router:   apiGroup,
		basePath: cfg.GetServerBaseApiPrefix(),
	}
}

func (s *HttpServer) Start() {
	serverAddr := fmt.Sprintf("%s:%s", s.cfg.GetServerHost(), s.cfg.GetServerPort())
	slog.Info("Starting HTTP server...", "address", serverAddr)

	go func() {
		if err := s.fiberApp.Listen(serverAddr); err != nil && err.Error() != "http: Server closed" {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	slog.Info("HTTP server started", "address", serverAddr)
}

func (s *HttpServer) Shutdown(ctx context.Context) {
	slog.Info("HTTP server shutting down")
	if err := s.fiberApp.ShutdownWithContext(ctx); err != nil {
		slog.Error("Error during HTTP server shutdown", "error", err)
		return
	}
	slog.Info("HTTP server shutdown gracefully")
}

func (s *HttpServer) GetRouter() fiber.Router {
	return s.router
}

func (s *HttpServer) GetApp() *fiber.App {
	return s.fiberApp
}
