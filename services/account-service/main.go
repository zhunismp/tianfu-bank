package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	. "github.com/zhunismp/tianfu-bank/services/account-service/adapter/primary/http"
	. "github.com/zhunismp/tianfu-bank/services/account-service/adapter/primary/http/account"
	. "github.com/zhunismp/tianfu-bank/services/account-service/adapter/secondary/infrastructure/config"
	. "github.com/zhunismp/tianfu-bank/services/account-service/adapter/secondary/infrastructure/database"
	. "github.com/zhunismp/tianfu-bank/services/account-service/adapter/secondary/repository/account"
	. "github.com/zhunismp/tianfu-bank/services/account-service/core/domain/account"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := LoadConfig()
	if err != nil {
		panic("ERROR: Failed to load config: " + err.Error())
	}

	db := NewPostgresDatabase(cfg)

	accountRepo := NewAccountRepository(db)
	accountSvc := NewAccountService(accountRepo)
	accountHttp := NewAccountHttpHandler(accountSvc)

	routeGroup := NewRouteGroup(accountHttp)

	httpServer := NewHttpServer(cfg)
	httpServer.SetUpRoute(routeGroup)
	httpServer.Start()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	httpServer.Shutdown(shutdownCtx)
}
