package main

import (
	"log"
	"encoding/json"

	"github.com/streadway/amqp"
	pod "github.com/icap-adaptation-service/pkg"
)

var exchange = "adaptation-exchange"
var routingKey = "adaptation-request"
var queueName = "adaptation-request-queue"

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq-service:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil,)
	failOnError(err, "Failed to declare an exchange")

	q, err := ch.QueueDeclare(queueName, false, false, false, false,	nil)
	failOnError(err, "Failed to declare a queue")

	err = ch.QueueBind(q.Name, routingKey, exchange, false, nil)
	failOnError(err, "Failed to bind queue")

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("received a message: %s", d.Body)
			var body map[string]interface{}

			err := json.Unmarshal(d.Body, &body)
			failOnError(err, "Failed to read the message body")

			fileID := body["file-id"].(string)
			input := body["source-file-location"].(string)
			output := body["rebuilt-file-location"].(string)

			podArgs, err := pod.NewPodArgs(fileID, input, output)
			failOnError(err, "Failed to initialize Pod")
			
			err = podArgs.CreatePod()
			failOnError(err, "Failed to start Pod")
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
