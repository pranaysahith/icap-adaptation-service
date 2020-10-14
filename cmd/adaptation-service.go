package main

// I'd recommend adding prometheus metrics to help with observability

import (
	"encoding/json"
	"log"

	pod "github.com/icap-adaptation-service/pkg"
	"github.com/streadway/amqp"
)

var exchange = "adaptation-exchange"
var routingKey = "adaptation-request"
var queueName = "adaptation-request-queue"

func main() {
	// hard-coded queue. should be env var
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq-service:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil)
	failOnError(err, "Failed to declare an exchange")

	q, err := ch.QueueDeclare(queueName, false, false, false, false, nil)
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
			// this is might be dangerous:
			// I am not sure about Rabbit semantics, but I suspect at least the current
			// mesasge (which is probably valid) might be lost if we fail to spawn a Pod
			// We should retry the k8s operation. Maybe we should put the message back?
			failOnError(err, "Failed to start Pod")
		}
	}()

	// this is dangerous: terminating the program with CTRL-C will immediately kill all running go-routines
	// potentially dropping messages on the floor.
	// Better: Use a close channel to indicate to the running goroutine that it should exit after finishing the current
	// message(s):
	// * Use sync.WaitGroup to track goroutines
	// * Instead of for d := range msgs, use select to read either from msgs or a stop channel
	//   Example: https://github.com/gargath/pleiades/blob/7e810c6d91e7b524ee09f6ab7afabf89ac3fb418/pkg/aggregator/kafka/aggregator.go#L81-L101
	// * Close the stop channel when CTRL-C is received and wait for the waitgroup before exiting
	//   Example: https://github.com/gargath/pleiades/blob/7e810c6d91e7b524ee09f6ab7afabf89ac3fb418/pkg/aggregator/kafka/aggregator.go#L127-L130
	//
	// To capture CTRL-C you can use something like this:
	//   quit := make(chan os.Signal, 1)
	//   signal.Notify(quit, os.Interrupt)
	//   go func() {
	// 	   <-quit
	// 	   aBlockingFunctionThatGracefullyShutsDownGoRoutines()
	//   }()

	log.Printf("[*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
