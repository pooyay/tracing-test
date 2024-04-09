package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"

	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	tracer     = otel.Tracer("rolldice")
	meter      = otel.Meter("rolldice")
	rollCnt    metric.Int64Counter
	messageCnt metric.Int64Counter
)

func init() {
	var err error
	rollCnt, err = meter.Int64Counter("dice.rolls",
		metric.WithDescription("The number of rolls by roll value"),
		metric.WithUnit("{roll}"))
	if err != nil {
		panic(err)
	}
}

func rolldice(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "roll")
	defer span.End()

	// parentFunction(ctx, w)

	roll := 1 + rand.Intn(6)

	rollValueAttr := attribute.Int("roll.value", roll)
	span.SetAttributes(rollValueAttr)
	rollCnt.Add(ctx, 1, metric.WithAttributes(rollValueAttr))

	resp := strconv.Itoa(roll) + "\n"
	if _, err := io.WriteString(w, resp); err != nil {
		log.Printf("Write failed: %v\n", err)
	}

	// publish - consume
	jsctx, js_span := tracer.Start(r.Context(), "jetstream")
	defer js_span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	nc, _ := nats.Connect(nats.DefaultURL)

	// Create a JetStream management interface
	js, _ := jetstream.New(nc)

	// Create a stream
	s, _ := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"ORDERS.*"},
	})

	// Publish some messages
	for i := 0; i < 10; i++ {
		js.Publish(ctx, "ORDERS.new", []byte("hello message "+strconv.Itoa(i)))
		fmt.Println("Published hello message", i)
		resp := "Published hello message" + strconv.Itoa(i) + "\n"
		if _, err := io.WriteString(w, resp); err != nil {
			log.Printf("Write failed: %v\n", err)
		}

	}

	// Create durable consumer
	c, _ := s.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "CONS",
		AckPolicy: jetstream.AckExplicitPolicy,
	})

	// Get 10 messages from the consumer
	messageCounter := 0
	msgs, _ := c.Fetch(10)
	for msg := range msgs.Messages() {
		msg.Ack()
		fmt.Println("Received a JetStream message via fetch: ", string(msg.Data()))

		messageCounter++
	}
	fmt.Printf("received %d messages\n", messageCounter)
	if msgs.Error() != nil {
		fmt.Println("Error during Fetch(): ", msgs.Error())
	}

	// Receive messages continuously in a callback
	// cons, _ := c.Consume(func(msg jetstream.Msg) {
	// 	msg.Ack()
	// 	fmt.Println("Received a JetStream message via callback: ", string(msg.Data()))
	// 	messageCounter++
	// })
	// defer cons.Stop()

	// Iterate over messages continuously
	// it, _ := c.Messages()
	// for i := 0; i < 10; i++ {
	// 	msg, _ := it.Next()
	// 	msg.Ack()
	// 	fmt.Println("Received a JetStream message via iterator: ", string(msg.Data()))
	// 	messageCounter++
	// }
	// it.Stop()

	// block until all 10 published messages have been processed
	// for messageCounter < 10 {
	// 	time.Sleep(10 * time.Millisecond)
	// }
	// nc.Close()

	messageCnteAttr := attribute.Int("messageCnt.value", messageCounter)
	span.SetAttributes(messageCnteAttr)
	rollCnt.Add(jsctx, 1, metric.WithAttributes(messageCnteAttr))

}
