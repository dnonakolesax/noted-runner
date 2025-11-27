package rabbit

import (
	"log/slog"

	"github.com/dnonakolesax/noted-runner/internal/logger"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitQueue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  *slog.Logger
	Queue   <-chan amqp.Delivery
}

func NewRabbit(address string, chanName string, rmqLogger *slog.Logger) (*RabbitQueue, error) {
	conn, err := amqp.Dial(address)
	if err != nil {
		rmqLogger.Error("unable to open connect to RabbitMQ server", logger.LogError(err))
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		rmqLogger.Error("failed to open a channel", logger.LogError(err))
		return nil, err
	}

	q, err := ch.QueueDeclare(
		chanName, // name
		false,    // durable
		false,    // delete when unused
		false,    // exclusive
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		rmqLogger.Error("failed to declare a queue", logger.LogError(err))
		return nil, err
	}

	messages, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		rmqLogger.Error("failed to register a consumer", logger.LogError(err))
		return nil, err
	}
	return &RabbitQueue{conn: conn, channel: ch, Queue: messages}, nil
}

func (rq *RabbitQueue) Close() {
	if rq.channel != nil {
		err := rq.channel.Close()
		if err != nil {
			rq.logger.Error("error closing rq chan", logger.LogError(err))
		}
	}
	if rq.conn != nil {
		err := rq.conn.Close()
		if err != nil {
			rq.logger.Error("error closing rq conn", logger.LogError(err))
		}
	}
}
