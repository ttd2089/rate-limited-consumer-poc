package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	consumer := newConsumer()
	handler := newHandler()

	for !isCancelled(ctx) {
		msg, err := consumer.Consume(ctx)
		if err != nil {
			fmt.Printf("error: consume: %v", err)
			<-time.After(5 * time.Second)
			continue
		}

		if err := handler.Handle(ctx, msg); err != nil {
			fmt.Printf("fatal: handle msg: %v", err)
			os.Exit(1)
			return
		}
	}
}

type message struct {
	customerID string
	type_      string
	body       string
}

type consumer interface {
	Consume(context.Context) (message, error)
}

type randomConsumer struct {
	customerIDs []string
	types       []string
}

func (r randomConsumer) Consume(_ context.Context) (message, error) {
	customerID := chooseRandom(r.customerIDs)
	type_ := chooseRandom(r.types)
	body := fmt.Sprintf("customer %q got a %q message at %v", customerID, type_, time.Now())
	return message{
		customerID: customerID,
		type_:      type_,
		body:       body,
	}, nil
}

func chooseRandom[T any](arr []T) T {
	return arr[rand.Int()%len(arr)]
}

func newConsumer() consumer {
	return randomConsumer{
		customerIDs: []string{
			"faa108f9-0815-4035-89c4-403b4f2f7948",
			"e62358f4-47bb-4a45-9db3-a1c5ad6cdab2",
			"139b70a3-60e8-47a0-9b7d-d8a369d18417",
			"432556b3-0a3b-4dbb-83fc-187115228f67",
		},
		types: []string{
			"foo",
			"bar",
			"baz",
		},
	}
}

type handler interface {
	Handle(context.Context, message) error
}

func newHandler() handler {
	return consoleHandler{}
}

type consoleHandler struct{}

func (c consoleHandler) Handle(_ context.Context, msg message) error {
	fmt.Printf("message: customer_id=%q type=%q body=%q\n", msg.customerID, msg.type_, msg.body)
	return nil
}

func isCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
