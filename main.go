package main

import (
	// "fmt"
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	// "github.com/nats-io/nats.go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	// "context"
	// "errors"
	// "log"
	// "net"
	// "net/http"
	// "os"
	// "os/signal"
	// "time"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
	// Connect to the NATS server
	// nc, err := nats.Connect(nats.DefaultURL)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer nc.Close()

	// // Create Jetstream context
	// js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// 	js, err := JetStreamInit()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	// js, err := jetstream.New(nc)
	// 	// if err != nil {
	// 	// 	log.Fatal(err)
	// 	// }

	// 	// Publisher
	// 	msg := []byte("Hello, Jetstream!")
	// 	_, err = js.Publish("subject", msg)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	// fmt.Printf("Published message with ID: %s\n", msgID)

	// 	// Wait for a short duration to ensure the message is processed
	// 	time.Sleep(500 * time.Millisecond)

	// 	// Consumer
	// 	sub, err := js.SubscribeSync("subject")
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	defer sub.Unsubscribe()

	// 	msgReceived, err := sub.NextMsg(time.Second)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	fmt.Printf("Received message: %s\n", string(msgReceived.Data))
	// }

	// func JetStreamInit() (nats.JetStreamContext, error) {
	// 	// Connect to NATS
	// 	nc, err := nats.Connect(nats.DefaultURL)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	// Create JetStream Context
	// 	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// return js, nil
}

func run() (err error) {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Set up OpenTelemetry.
	otelShutdown, err := setupOTelSDK(ctx)
	if err != nil {
		return
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// Start HTTP server.
	srv := &http.Server{
		Addr:         ":8081",
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHTTPHandler(),
	}
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		// Error when starting HTTP server.
		return
	case <-ctx.Done():
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	// When Shutdown is called, ListenAndServe immediately returns ErrServerClosed.
	err = srv.Shutdown(context.Background())
	return
}

func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// handleFunc is a replacement for mux.HandleFunc
	// which enriches the handler's HTTP instrumentation with the pattern as the http.route.
	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
		// Configure the "http.route" for the HTTP instrumentation.
		handler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
		mux.Handle(pattern, handler)
	}

	// Register handlers.
	handleFunc("/rolldice", rolldice)
	// handleFunc("/waiting", waiting)

	// Add HTTP instrumentation for the whole server.
	handler := otelhttp.NewHandler(mux, "/")
	return handler
}
