package main

import (
	"log"
	"github.com/streadway/amqp"
	"encoding/json"
	"github.com/icap-adaptation-service/pkg"
)

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare("hello", false, false, false, false,	nil)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("received a message: %s", d.Body)
			var body map[string]interface{}

			err := json.Unmarshal(d.Body, &body)
			failOnError(err, "Failed to read the message body")

			podArgs := PodArgs{
				FileID: body["file-id"].(string),
				Input: body["source-file-location"].(string),
				Output: body["rebuilt-file-location"].(string),
			}

			podArgs.Create()
		}
	}()

	log.Printf("[*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
