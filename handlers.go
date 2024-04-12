package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"time"

	"go.opentelemetry.io/otel"
)

var (
	tracer = otel.Tracer("publish-consume")
	meter  = otel.Meter("publish-consume")
)

func publish(w http.ResponseWriter, r *http.Request) {
	Nc.mu.Lock()
	_, span := tracer.Start(r.Context(), "publish")
	defer span.End()

	nc := Nc.nc

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	js, err := initJetStream(nc)
	if err != nil {
		log.Fatal(err)
	}

	js.Publish(ctx, "ORDERS.new", []byte("hello message"))
	fmt.Println("message published.")
	Nc.mu.Unlock()
}

func ConsumerJob(ctx context.Context) {
	Nc.mu.Lock()
	_, span := tracer.Start(ctx, "consumer")
	defer span.End()

	nc := Nc.nc

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
			Nc.mu.Unlock()
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
