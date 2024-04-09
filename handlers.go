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

func connection() (*nats.Conn, error) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	return nc, err
}

func publish(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), "jetstream")
	defer span.End()

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	nc, err := connection()
	if err != nil {
		log.Fatal(err)
	}

	// Create a JetStream management interface
	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal(err)
	}

	js.Publish(ctx, "ORDERS.new", []byte("hello message"))
	fmt.Println("message published.")

	// for i := 0; i < 10; i++ {
	// 	js.Publish(ctx, "ORDERS.new", []byte("hello message "+strconv.Itoa(i)))
	// 	fmt.Println("Published hello message", i)
	// 	resp := "Published hello message" + strconv.Itoa(i) + "\n"
	// 	if _, err := io.WriteString(w, resp); err != nil {
	// 		log.Printf("Write failed: %v\n", err)
	// 	}
	// }
}

func consume(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), "jetstream")
	defer span.End()

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	nc, err := connection()
	if err != nil {
		log.Fatal(err)
	}

	// Create a JetStream management interface
	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal(err)
	}

	// Create a stream
	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"ORDERS.*"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create durable consumer
	c, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "CONS",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		log.Fatal(err)
	}

	msg, _ := c.Next()
	if err != nil {
		log.Fatal(err)
	}
	msg.Ack()
	fmt.Println("Recieved message: ", string(msg.Data()))
}

func ConsumerJob(ctx context.Context) {
	nc, err := connection()
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// Create a JetStream management interface
	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal(err)
	}

	// Create a stream
	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"ORDERS.*"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create durable consumer
	c, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "CONS",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
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
