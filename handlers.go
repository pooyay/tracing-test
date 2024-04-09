package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
)

var (
	tracer = otel.Tracer("publish-consume")
	meter  = otel.Meter("publish-consume")
)

func publish(w http.ResponseWriter, r *http.Request) {
	_, js_span := tracer.Start(r.Context(), "jetstream")
	defer js_span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	nc, _ := nats.Connect(nats.DefaultURL)

	// Create a JetStream management interface
	js, _ := jetstream.New(nc)

	for i := 0; i < 10; i++ {
		js.Publish(ctx, "ORDERS.new", []byte("hello message "+strconv.Itoa(i)))
		fmt.Println("Published hello message", i)
		resp := "Published hello message" + strconv.Itoa(i) + "\n"
		if _, err := io.WriteString(w, resp); err != nil {
			log.Printf("Write failed: %v\n", err)
		}
	}
}

func consume(w http.ResponseWriter, r *http.Request) {
	_, js_span := tracer.Start(r.Context(), "jetstream")
	defer js_span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	nc, _ := nats.Connect(nats.DefaultURL)

	// Create a JetStream management interface
	js, _ := jetstream.New(nc)

	// Create a stream
	stream, _ := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"ORDERS.*"},
	})

	// Create durable consumer
	c, _ := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "CONS",
		AckPolicy: jetstream.AckExplicitPolicy,
	})

	// Get 10 messages from the consumer
	messageCounter := 0
	msgs, _ := c.Fetch(10)
	for msg := range msgs.Messages() {
		msg.Ack()
		fmt.Println("Received a JetStream message via fetch: ", string(msg.Data()))
		resp := "Received a JetStream message via fetch:" + string(msg.Data()) + "\n"
		if _, err := io.WriteString(w, resp); err != nil {
			log.Printf("Write failed: %v\n", err)
		}

		messageCounter++
	}
	fmt.Printf("received %d messages\n", messageCounter)
	resp := "received " + strconv.Itoa(messageCounter) + " messages" + "\n"
	if _, err := io.WriteString(w, resp); err != nil {
		log.Printf("Write failed: %v\n", err)
	}
	if msgs.Error() != nil {
		fmt.Println("Error during Fetch(): ", msgs.Error())
	}
}
