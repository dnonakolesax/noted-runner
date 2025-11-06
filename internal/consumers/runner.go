package consumers

import (
	"encoding/json"
	"log/slog"

	compilerDelivery "github.com/dnonakolesax/noted-runner/internal/delivery/compiler/v1/http"
	"github.com/dnonakolesax/noted-runner/internal/model"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RunnerConsumer struct {
	messages  <-chan amqp.Delivery
	delivery *compilerDelivery.ComilerDelivery
}

func NewRunnerConsumer(messages <-chan amqp.Delivery, delivery *compilerDelivery.ComilerDelivery) *RunnerConsumer {
	return &RunnerConsumer{messages: messages, delivery: delivery}
}

func (rc *RunnerConsumer) Consume() {
	for msg := range rc.messages {
		var kmessage model.KernelMessage
		err := json.Unmarshal(msg.Body, &kmessage)

		if err != nil {
			slog.Error("error unmarshaling kernel data", slog.String("error", err.Error()))
			continue
		}

		rc.delivery.SendMemes(kmessage.KernelID, string(msg.Body))
	}
}

