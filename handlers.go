package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var (
	tracer = otel.Tracer("publish-consume")
	meter  = otel.Meter("publish-consume")
)

type server struct {
	nc *nats.Conn
}

func routes(nc *nats.Conn) http.Handler {
	mux := http.NewServeMux()

	// handleFunc is a replacement for mux.HandleFunc
	// which enriches the handler's HTTP instrumentation with the pattern as the http.route.
	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
		// Configure the "http.route" for the HTTP instrumentation.
		handler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
		mux.Handle(pattern, handler)
	}

	s := server{nc: nc}

	// Register handlers.
	handleFunc("/publish", s.publish)

	// Add HTTP instrumentation for the whole server.
	handler := otelhttp.NewHandler(mux, "/")
	return handler
}

func (s *server) publish(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "publish")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	js, err := initJetStream(s.nc)
	if err != nil {
		log.Fatal(err)
	}

	js.Publish(ctx, "ORDERS.new", []byte("hello message"))
	fmt.Println("message published.")
}
