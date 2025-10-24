package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

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

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<- ch

	fmt.Println("Server shutting down...")
}
