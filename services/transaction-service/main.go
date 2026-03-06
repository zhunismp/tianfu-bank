package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhunismp/tianfu-bank/shared/messaging"

	httpAdapter "github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/primary/http"
	txnHandler "github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/primary/http/transaction"
	mqConsumer "github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/primary/mq/account"
	cfgAdapter "github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/secondary/infrastructure/config"
	dbAdapter "github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/secondary/infrastructure/database"
	mqPublisher "github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/secondary/messaging/rabbitmq"
	eventStoreRepo "github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/secondary/repository/event_store"
	snapshotRepo "github.com/zhunismp/tianfu-bank/services/transaction-service/adapter/secondary/repository/snapshot"
	"github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Config ---
	cfg, err := cfgAdapter.LoadConfig()
	if err != nil {
		panic("ERROR: Failed to load config: " + err.Error())
	}

	// --- Database ---
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

	// --- Repositories ---
	eventStore := eventStoreRepo.NewEventStoreRepository(db)
	snapshot := snapshotRepo.NewSnapshotRepository(db)
	txProvider := dbAdapter.NewTransactionProvider(db)

	// --- Publisher ---
	publisher := mqPublisher.NewBalancePublisher(rmqCh)

	// --- Service ---
	txnSvc := transaction.NewTransactionService(txProvider, publisher)

	// --- HTTP ---
	txnHttpHandler := txnHandler.NewTransactionHttpHandler(txnSvc)
	routeGroup := httpAdapter.NewRouteGroup(txnHttpHandler)

	httpServer := httpAdapter.NewHttpServer(cfg)
	httpServer.SetUpRoute(routeGroup)
	httpServer.Start()

	// --- MQ Consumer ---
	consumer := mqConsumer.NewAccountCreatedConsumer(rmqCh, snapshot, eventStore)
	if err := consumer.Start(ctx); err != nil {
		panic("ERROR: Failed to start AccountCreated consumer: " + err.Error())
	}

	// --- Wait for shutdown ---
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	httpServer.Shutdown(shutdownCtx)
	messaging.CloseRabbitMQ(rmqConn, rmqCh)
	dbAdapter.ShutdownDatabase(db)

	slog.Info("Transaction service shutdown gracefully")
}
