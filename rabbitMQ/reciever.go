package rabbitMQ

import (
	"api-service/cake_websocket"

	"github.com/streadway/amqp"
)

func RunReciever(hub *cake_websocket.Hub) {

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		PrintError(err, "Failed to connect to RabbitMQ in reciever:")
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		PrintError(err, "Failed to open a channel in reciever:")
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		PrintError(err, "Failed to declare a queue in reciever:")
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		PrintError(err, "Failed to register a consumer in reciever:")
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			// log.Printf("Received a message: %s", d.Body)
			hub.SendMessage(string(d.Body))
		}
	}()

	<-forever
}
