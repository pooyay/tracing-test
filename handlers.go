package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
)

var (
	tracer = otel.Tracer("publish-consume")
	meter  = otel.Meter("publish-consume")
)

func initConnection() (*nats.Conn, jetstream.JetStream) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	// Create a JetStream management interface
	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal(err)
	}
	return nc, js
}

func publish(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), "jetstream")
	defer span.End()

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	nc, js := initConnection()
	defer nc.Close()

	js.Publish(ctx, "ORDERS.new", []byte("hello message"))
	fmt.Println("message published.")
}

func createStream(ctx context.Context, js jetstream.JetStream) (jetstream.Stream, error) {
	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"ORDERS.*"},
	})
	return stream, err
}

func createDurableConsumer(ctx context.Context, stream jetstream.Stream) (jetstream.Consumer, error) {
	c, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "CONS",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	return c, err
}

func ConsumerJob(ctx context.Context) {
	nc, js := initConnection()
	defer nc.Close()

	// Create a stream
	stream, err := createStream(ctx, js)
	if err != nil {
		log.Fatal(err)
	}

	// Create durable consumer
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

			// Process the message
			msg.Ack()
			fmt.Println("Received a JetStream message: ", string(msg.Data()))
		}
	}
}
