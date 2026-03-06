package http

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/primary/http/transaction"
)

type RouteGroup struct {
	transaction *transaction.TransactionHttpHandler
}

func NewRouteGroup(txn *transaction.TransactionHttpHandler) *RouteGroup {
	return &RouteGroup{transaction: txn}
}

func (s *HttpServer) SetUpRoute(routeGroup *RouteGroup) {
	txn := routeGroup.transaction

	s.registerAPIGroup("/transactions", func(txnRouter fiber.Router) {
		txnRouter.Post("/deposit", txn.Deposit)
		txnRouter.Post("/withdraw", txn.Withdraw)
		txnRouter.Post("/transfer", txn.Transfer)
		txnRouter.Get("/history/:accountId", txn.GetHistory)
	})

	s.fiberApp.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	slog.Info("HTTP routes setup successfully")
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
