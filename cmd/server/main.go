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
		log.Fatal("Failed to connect to rabbitmq")
	}
	defer conn.Close()

	fmt.Println("Successfully connected to rabbitmq")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Failed to create channel")
	}

	err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
	if err != nil {
		log.Fatalln("Failed to publish pause")
	}

	_, _, err =pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		fmt.Sprintf("%s.*", routing.GameLogSlug),
		pubsub.QueueTypeDurable,
	)
	if err != nil {
		log.Fatalln("Failed to declare and bind to topic exchange")
	}

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
				log.Fatalln("Failed to publish pause")
			}
			fmt.Println("Pause message sent!")

		case "resume":
			fmt.Println("Sending resume message...")
			state := routing.PlayingState{IsPaused: false}
			err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, state)
			if err != nil {
				log.Fatalln("Failed to publish pause")
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
