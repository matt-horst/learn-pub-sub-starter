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
		pubsub.Transient,
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
		pubsub.Transient,
		handlerArmyMove(gameState, ch),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to army_moves: %v\n", err)
	}

	// Subscribe to handle war
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix + ".*",
		pubsub.Durable,
		handlerWarMessages(gameState),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to war: %v\n", err)
	}

	done := false
	for !done {
		input := gamelogic.GetInput()

		if len(input) < 1 {
			continue
		}

		cmd := input[0]
		switch cmd {
		case "spawn": err := gameState.CommandSpawn(input)
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

func handlerPause(gameState *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(playingState routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")

		gameState.HandlePause(playingState)

		return pubsub.Ack
	}
}

func handlerArmyMove(gameState *gamelogic.GameState, ch *ampq.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")

		result := gameState.HandleMove(move)
		switch result {
		case gamelogic.MoveOutComeSafe, gamelogic.MoveOutcomeMakeWar:
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, gameState.Player.Username),
				gamelogic.RecognitionOfWar{ Attacker: move.Player, Defender: gameState.Player},
			)
			return pubsub.NackRequeue
		default:
			return pubsub.NackDiscard
		}
	}
}

func handlerWarMessages(gameState *gamelogic.GameState) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")

		outcome, _, _ := gameState.HandleWar(rw)

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue

		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard

		case gamelogic.WarOutcomeOpponentWon:
			return pubsub.Ack

		case gamelogic.WarOutcomeYouWon:
			return pubsub.Ack

		case gamelogic.WarOutcomeDraw:
			return pubsub.Ack

		default:
			log.Printf("Unrecognized war outcome")
			return pubsub.NackDiscard
		}
	}
}
