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
		log.Fatalf("Failed to connect to rabbitmq: %v\n", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel to rabbitmq: %v\n", err)
	}

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Failed to get valid username: %v\n", err)
	}


	gameState := gamelogic.NewGameState(userName)

	// Subscribe to pause
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, userName),
		routing.PauseKey,
		pubsub.QueueTypeTransient,
		handlerPause(gameState),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to pause: %v\n", err)
	}

	// Subscribe to army moves
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, userName),
		fmt.Sprintf("%s.*", routing.ArmyMovesPrefix),
		pubsub.QueueTypeTransient,
		handlerArmyMove(gameState),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to army_moves: %v\n", err)
	}

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
			move, err := gameState.CommandMove(input)
			if err != nil {
				fmt.Printf("Couldn't move units: %v\n", err)
				continue
			}

			err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, userName),
				move,
			)
			if err != nil {
				fmt.Printf("Couldn't move units: %v\n", err)
				continue
			}

			fmt.Println("Move published successfully!")

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

func handlerPause(gameState *gamelogic.GameState) func(routing.PlayingState) {
	return func(playingState routing.PlayingState) {
		defer fmt.Print("> ")

		gameState.HandlePause(playingState)
	}
}

func handlerArmyMove(gameState *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(move gamelogic.ArmyMove) {
		defer fmt.Print("> ")

		gameState.HandleMove(move)
	}
}
