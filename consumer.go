package main

import (
	"context"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

func consumerJob(ctx context.Context, nc *nats.Conn) {
	ctx, span := tracer.Start(ctx, "consumer")
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
				log.Fatalf("cant get the message: %v", err)
			}

			msg.Ack()
			fmt.Println("Received a JetStream message: ", string(msg.Data()))
		}
	}
}
