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

func initConnection() (*nats.Conn, error) {
	nc, err := nats.Connect(nats.DefaultURL)
	return nc, err
}

// Create a JetStream management interface
func initJetStream(nc *nats.Conn) (jetstream.JetStream, error) {
	js, err := jetstream.New(nc)
	return js, err
}

func publish(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), "publish")
	defer span.End()

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	nc, err := initConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	js, err := initJetStream(nc)
	if err != nil {
		log.Fatal(err)
	}

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
	_, span := tracer.Start(ctx, "consumer")
	defer span.End()

	nc, err := initConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

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
