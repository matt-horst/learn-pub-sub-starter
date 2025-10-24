package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"


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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<- sigCh

	fmt.Println("Server shutting down...")
}
