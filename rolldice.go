package main

import (
	"context"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	tracer  = otel.Tracer("rolldice")
	meter   = otel.Meter("rolldice")
	rollCnt metric.Int64Counter
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

	parentFunction(ctx, w)

	roll := 1 + rand.Intn(6)

	rollValueAttr := attribute.Int("roll.value", roll)
	span.SetAttributes(rollValueAttr)
	rollCnt.Add(ctx, 1, metric.WithAttributes(rollValueAttr))

	resp := strconv.Itoa(roll) + "\n"
	if _, err := io.WriteString(w, resp); err != nil {
		log.Printf("Write failed: %v\n", err)
	}

}

func parentFunction(ctx context.Context, w http.ResponseWriter) {
	ctx, parentSpan := tracer.Start(ctx, "parent")
	defer parentSpan.End()

	childFunction(ctx, w)

	io.WriteString(w, "Waiting for 1 second in parent...\n")
	time.Sleep(1 * time.Second)
	io.WriteString(w, "Waited for 1 seconds in parent.\n")
}

func childFunction(ctx context.Context, w http.ResponseWriter) {
	ctx, childSpan := tracer.Start(ctx, "child")
	defer childSpan.End()

	io.WriteString(w, "Waiting for 500 milisecond in child...\n")
	time.Sleep(500 * time.Millisecond)
	io.WriteString(w, "Waited for 500 miliseconds in child.\n")
}
