package messaging

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/events"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	ch        *amqp.Channel
	queueName string
}

func NewRabbitMQPublisher(url, queueName string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": "payment.dlq",
		},
	)
	if err != nil {
		return nil, err
	}

	return &RabbitMQPublisher{
		ch:        ch,
		queueName: queueName,
	}, nil
}

func (r *RabbitMQPublisher) PublishPaymentCompleted(ctx context.Context, event events.PaymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return r.ch.PublishWithContext(ctx,
		"",
		r.queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
}
