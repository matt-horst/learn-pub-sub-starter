package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	ampq "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	QueueTypeDurable SimpleQueueType = iota
	QueueTypeTransient
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

func DeclareAndBind(
	conn *ampq.Connection,
	exchange string,
	queueName string,
	key string,
	queueType SimpleQueueType,
) (*ampq.Channel, ampq.Queue, error){
	ch, err := conn.Channel()
	if err != nil {
		return nil, ampq.Queue{}, fmt.Errorf("Failed to create channel: %v", err)
	}

	isDurable := queueType == QueueTypeDurable
	isAutoDelete := queueType == QueueTypeTransient
	isExclusive := queueType == QueueTypeTransient
	q, err := ch.QueueDeclare(queueName, isDurable, isAutoDelete, isExclusive, false, nil)
	if err != nil {
		return nil, ampq.Queue{}, fmt.Errorf("Failed to declare queue: %v", err)
	}

	return ch, q, nil
}
