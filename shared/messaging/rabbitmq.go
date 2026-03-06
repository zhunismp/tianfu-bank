package messaging

import (
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQConfig holds connection parameters for RabbitMQ.
type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

// DSN returns the AMQP connection string.
func (c *RabbitMQConfig) DSN() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/", c.User, c.Password, c.Host, c.Port)
}

const (
	ExchangeName = "tianfu.events"
	ExchangeKind = "topic"

	RoutingKeyAccountCreated = "account.created"
	RoutingKeyBalanceUpdated = "balance.updated"
)

// ConnectRabbitMQ establishes a connection and channel to RabbitMQ,
// and declares the shared topic exchange.
func ConnectRabbitMQ(cfg *RabbitMQConfig) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(cfg.DSN())
	if err != nil {
		return nil, nil, fmt.Errorf("rabbitmq dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("rabbitmq channel: %w", err)
	}

	// Declare the shared exchange
	if err := ch.ExchangeDeclare(
		ExchangeName,
		ExchangeKind,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, fmt.Errorf("rabbitmq exchange declare: %w", err)
	}

	slog.Info("RabbitMQ connected and exchange declared", "exchange", ExchangeName)
	return conn, ch, nil
}

// CloseRabbitMQ gracefully closes a channel and connection.
func CloseRabbitMQ(conn *amqp.Connection, ch *amqp.Channel) {
	if ch != nil {
		if err := ch.Close(); err != nil {
			slog.Error("Failed to close RabbitMQ channel", "error", err)
		}
	}
	if conn != nil {
		if err := conn.Close(); err != nil {
			slog.Error("Failed to close RabbitMQ connection", "error", err)
		}
	}
	slog.Info("RabbitMQ connection closed")
}
