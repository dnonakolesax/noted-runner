package consumers

import (
	"encoding/json"
	"log/slog"

	compilerDelivery "github.com/dnonakolesax/noted-runner/internal/delivery/compiler/v1/http"
	"github.com/dnonakolesax/noted-runner/internal/logger"
	"github.com/dnonakolesax/noted-runner/internal/model"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RunnerConsumer struct {
	messages  <-chan amqp.Delivery
	delivery *compilerDelivery.ComilerDelivery
	logger *slog.Logger
}

func NewRunnerConsumer(messages <-chan amqp.Delivery, delivery *compilerDelivery.ComilerDelivery, logger *slog.Logger) *RunnerConsumer {
	return &RunnerConsumer{messages: messages, delivery: delivery, logger: logger}
}

func (rc *RunnerConsumer) Consume() {
	for msg := range rc.messages {
		rc.logger.Info("received rmq message")
		var kmessage model.KernelMessage
		err := json.Unmarshal(msg.Body, &kmessage)

		if err != nil {
			rc.logger.Error("error unmarshaling kernel data", logger.LogError(err))
			continue
		}

		rc.delivery.SendMemes(kmessage.KernelID, string(msg.Body))
	}
}
