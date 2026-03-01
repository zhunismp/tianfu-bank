package http

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/zhunismp/tianfu-bank/services/account-service/adapter/primary/http/account"
)

type RouteGroup struct {
	account *account.AccountHttpHandler
}

func NewRouteGroup(account *account.AccountHttpHandler) *RouteGroup {
	return &RouteGroup{
		account: account,
	}
}

func (s *HttpServer) SetUpRoute(routeGroup *RouteGroup) {
	account := routeGroup.account

	s.registerAPIGroup("/accounts", func(accountRouter fiber.Router) {
		accountRouter.Post("/", account.CreateAccount)
		accountRouter.Get("/:accountId", account.GetAccount)
	})

	s.fiberApp.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	slog.Info("HTTP routes was setup successfully")
}

func (s *HttpServer) registerAPIGroup(subPrefix string, groupRegistrar func(router fiber.Router)) {
	if !strings.HasPrefix(subPrefix, "/") && subPrefix != "" {
		subPrefix = "/" + subPrefix
	}
	fullPrefix := s.basePath + subPrefix
	if s.basePath == "/" && subPrefix == "" {
		fullPrefix = "/"
	} else if s.basePath == "/" && strings.HasPrefix(subPrefix, "/") {
		fullPrefix = subPrefix
	} else if s.basePath != "" && subPrefix == "" {
		fullPrefix = s.basePath
	}

	group := s.router.Group(subPrefix)
	groupRegistrar(group)
	slog.Info("Registered API group", "fullPrefix", fullPrefix)
}
