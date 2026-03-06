package account

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// AMQPChannel is a testable subset of *amqp.Channel used by the consumer.
type AMQPChannel interface {
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
}
