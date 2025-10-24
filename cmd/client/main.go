package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	ampq "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	url := "amqp://guest:guest@localhost:5672/"
	conn, err := ampq.Dial(url)
	if err != nil {
		log.Fatalln("Failed to connect to rabbitmq")
	}

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalln("Failed to get valid username")
	}

	_, _, err = pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, userName),
		routing.PauseKey,
		pubsub.QueueTypeTransient,
	)

	if err != nil {
		log.Fatalln("Failed to declare and bind transient queue")
	}

	gameState := gamelogic.NewGameState(userName)

	done := false
	for !done {
		input := gamelogic.GetInput()

		if len(input) < 1 {
			continue
		}

		cmd := input[0]
		switch cmd {
		case "spawn":
			err := gameState.CommandSpawn(input)
			if err != nil {
				fmt.Printf("Couldn't spawn unit: %v\n", err)
				continue
			}

		case "move":
			_, err := gameState.CommandMove(input)
			if err != nil {
				fmt.Printf("Couldn't move units%v\n", err)
				continue
			}

		case "status":
			gameState.CommandStatus()

		case "help":
			gamelogic.PrintClientHelp()

		case "spam":
			fmt.Println("Spamming not allowed yet!")

		case "quit":
			gamelogic.PrintQuit()
			done = true
		default:
			fmt.Printf("Unrecognized command: %v\n", cmd)
		}
	}

	fmt.Println("Server shutting down...")
}
