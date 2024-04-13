package main

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var nc *nats.Conn

func initConnection() (*nats.Conn, error) {
	nc, err := nats.Connect(nats.DefaultURL)
	return nc, err
}

func assignConnection(newConnection *nats.Conn) {
	nc = newConnection
}

// Create a JetStream management interface
func initJetStream(nc *nats.Conn) (jetstream.JetStream, error) {
	js, err := jetstream.New(nc)
	return js, err
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
