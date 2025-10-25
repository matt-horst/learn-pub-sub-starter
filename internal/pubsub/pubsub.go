package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"

	ampq "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int
const (
	Durable SimpleQueueType = iota
	Transient
)

type AckType int
const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func PublishGob[T any] (ch *ampq.Channel, exchange, key string, val T) error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(val)
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
			ContentType: "application/gob",
			Body: buf.Bytes(),
		},
	)
	if err != nil {
		return fmt.Errorf("Failed to publish: %v", err)
	}

	return nil
}

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

	q, err := ch.QueueDeclare(
		queueName,
		queueType == Durable,
		queueType != Durable,
		queueType != Durable,
		false,
		ampq.Table{"x-dead-letter-exchange": "peril_dlx"},
	)
	if err != nil {
		return nil, ampq.Queue{}, fmt.Errorf("Failed to declare queue: %v", err)
	}

	err = ch.QueueBind(q.Name, key, exchange, false, nil)
	if err != nil {
		return nil, ampq.Queue{}, fmt.Errorf("Failed to bind queue: %v", err)
	}

	return ch, q, nil
}

func SubscribeGob[T any] (
	conn *ampq.Connection,
	exchange string,
	queueName string,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(
		conn,
		exchange,
		queueName,
		key,
		queueType,
		handler,
		func(b []byte) (T, error) {
			buf := bytes.NewBuffer(b)
			decoder := gob.NewDecoder(buf)
			var t T
			err := decoder.Decode(&t)

			return t, err
		},
	)
}

func SubscribeJSON[T any] (
	conn *ampq.Connection,
	exchange string,
	queueName string,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(
		conn,
		exchange,
		queueName,
		key,
		queueType,
		handler,
		func(b []byte) (T, error) {
			var t T
			err := json.Unmarshal(b, &t)
			return t, err
		},
	)
}

func subscribe[T any](
	conn *ampq.Connection,
	exchange string,
	queueName string,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("Failed to declare and bind: %v", err)
	}

	ds, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("Failed to consume queue: %v", err)
	}

	go func() {
		for d := range ds {
			t, err := unmarshaller(d.Body)
			if err != nil {
				log.Printf("Failed to decode json: %v", err)
				continue
			}

			ack := handler(t)

			switch ack {
			case Ack:
				err = d.Ack(false)
				if err != nil {
					log.Printf("Failed to ack delivery: %v", err)
					continue
				}

			case NackRequeue:
				err = d.Nack(false, true)
				if err != nil {
					log.Printf("Failed to nack delivery: %v", err)
					continue
				}

			case NackDiscard:
				err = d.Nack(false, false)
				if err != nil {
					log.Printf("Failed to nack delivery: %v", err)
					continue
				}

			}
		}
	} ()

	return nil
}
