package main

import (
	"context"
	"fmt"
	"log"
)

func consumerJob(ctx context.Context) {
	_, span := tracer.Start(ctx, "consumer")
	defer span.End()

	js, err := initJetStream(nc)
	if err != nil {
		log.Fatal(err)
	}

	stream, err := createStream(ctx, js)
	if err != nil {
		log.Fatal(err)
	}

	c, err := createDurableConsumer(ctx, stream)
	if err != nil {
		log.Fatal(err)
	}

	// Consume messages continuously
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get the message from the consumer
			msg, err := c.Next()
			if err != nil {
				log.Fatal(err)
			}

			msg.Ack()
			fmt.Println("Received a JetStream message: ", string(msg.Data()))
		}
	}
}
