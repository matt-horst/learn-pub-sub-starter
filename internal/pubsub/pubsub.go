package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	ampq "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any] (ch *ampq.Channel, exchange, key string, val T) error {
	json, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("Failed to marshal object: %v", err)
	}

	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		ampq.Publishing{
			ContentType: "application/json",
			Body: json,
		},
	)
	if err != nil {
		return fmt.Errorf("Failed to publish: %v", err)
	}

	return nil
}
