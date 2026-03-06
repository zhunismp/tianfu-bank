package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// AMQPChannel is a testable subset of *amqp.Channel used by the publisher.
type AMQPChannel interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}
