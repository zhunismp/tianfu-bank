package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	httpAdapter "github.com/zhunismp/tianfu-bank/services/account-service/adapter/primary/http"
	"github.com/zhunismp/tianfu-bank/services/account-service/adapter/primary/http/account"
	mqConsumer "github.com/zhunismp/tianfu-bank/services/account-service/adapter/primary/mq/account"
	cfgAdapter "github.com/zhunismp/tianfu-bank/services/account-service/adapter/secondary/infrastructure/config"
	dbAdapter "github.com/zhunismp/tianfu-bank/services/account-service/adapter/secondary/infrastructure/database"
	mqPublisher "github.com/zhunismp/tianfu-bank/services/account-service/adapter/secondary/messaging/rabbitmq"
	accountRepo "github.com/zhunismp/tianfu-bank/services/account-service/adapter/secondary/repository/account"
	domain "github.com/zhunismp/tianfu-bank/services/account-service/core/domain/account"
	"github.com/zhunismp/tianfu-bank/shared/messaging"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := cfgAdapter.LoadConfig()
	if err != nil {
		panic("ERROR: Failed to load config: " + err.Error())
	}

	db := dbAdapter.NewPostgresDatabase(cfg)

	// --- RabbitMQ ---
	rmqCfg := &messaging.RabbitMQConfig{
		Host:     cfg.GetRabbitMQHost(),
		Port:     cfg.GetRabbitMQPort(),
		User:     cfg.GetRabbitMQUser(),
		Password: cfg.GetRabbitMQPassword(),
	}
	rmqConn, rmqCh, err := messaging.ConnectRabbitMQ(rmqCfg)
	if err != nil {
		panic("ERROR: Failed to connect to RabbitMQ: " + err.Error())
	}

	// --- Repositories & Publisher ---
	repo := accountRepo.NewAccountRepository(db)
	publisher := mqPublisher.NewAccountPublisher(rmqCh)

	// --- Service ---
	accountSvc := domain.NewAccountService(repo, publisher)
	accountHttp := account.NewAccountHttpHandler(accountSvc)

	routeGroup := httpAdapter.NewRouteGroup(accountHttp)

	httpServer := httpAdapter.NewHttpServer(cfg)
	httpServer.SetUpRoute(routeGroup)
	httpServer.Start()

	// --- MQ Consumer ---
	consumer := mqConsumer.NewBalanceUpdatedConsumer(rmqCh, repo)
	if err := consumer.Start(ctx); err != nil {
		panic("ERROR: Failed to start BalanceUpdated consumer: " + err.Error())
	}

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	httpServer.Shutdown(shutdownCtx)
	messaging.CloseRabbitMQ(rmqConn, rmqCh)
	dbAdapter.ShutdownDatabase(db)

	slog.Info("Application shutdown gracefully")
}
