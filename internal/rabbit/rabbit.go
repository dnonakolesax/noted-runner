package rabbit

import (
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitQueue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	Queue   <-chan amqp.Delivery
}

func NewRabbit(address string, chanName string) (*RabbitQueue, error) {
	conn, err := amqp.Dial(address)
	if err != nil {
		slog.Error("unable to open connect to RabbitMQ server", slog.String("error", err.Error()))
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		slog.Error("failed to open a channel", slog.String("error", err.Error()))
		return nil, err
	}

	q, err := ch.QueueDeclare(
		chanName, // name
		false,           // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		slog.Error("failed to declare a queue", slog.String("error", err.Error()))
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
		slog.Error("failed to register a consumer", slog.String("error", err.Error()))
		return nil, err
	}
	return &RabbitQueue{conn: conn, channel: ch, Queue: messages}, nil
}

func (rq *RabbitQueue) Close() {
	if rq.channel != nil {
		_ = rq.channel.Close()
	}
	if rq.conn != nil {
		_ = rq.conn.Close()
	}
}
