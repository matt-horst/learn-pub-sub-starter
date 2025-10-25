package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	url := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Failed to connect to rabbitmq: %v\n", err)
	}
	defer conn.Close()

	fmt.Println("Successfully connected to rabbitmq")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %v\n", err)
	}

	pubsub.SubscribeGob(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		fmt.Sprintf("%s.*", routing.GameLogSlug),
		pubsub.Durable,
		handlerGameLog,
	)

	gamelogic.PrintServerHelp()

	done := false
	for !done {
		input := gamelogic.GetInput()
		if len(input) < 1 {
			continue
		}

		switch input[0] {
		case "pause":
			fmt.Println("Sending pause message...")
			state := routing.PlayingState{IsPaused: true}
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, state)
			if err != nil {
				log.Fatalf("Failed to publish pause: %v\n", err)
			}
			fmt.Println("Pause message sent!")

		case "resume":
			fmt.Println("Sending resume message...")
			state := routing.PlayingState{IsPaused: false}
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, state)
			if err != nil {
				log.Fatalf("Failed to publish pause: %v\n", err)
			}
			fmt.Println("Resume message sent!")
		case "quit":
			fmt.Println("Exiting...")
			done = true

		default:
			fmt.Printf("Unrecognized command: %v\n", input[0])
		}
	}

	fmt.Println("Server shutting down...")
}

func handlerGameLog(gl routing.GameLog) pubsub.AckType {
	defer fmt.Print("> ")

	err := gamelogic.WriteLog(gl)
	if err != nil {
		log.Printf("Failed to write log: %v\n", err)
		return pubsub.NackRequeue
	}

	return pubsub.Ack
}
