package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	consumer, err := newConsumer()
	if err != nil {
		fmt.Printf("fatal: %v\n", err)
		os.Exit(1)
		return
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			fmt.Printf("error: close consumer: %v\n", err)
		}
	}()

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

		if err := consumer.Commit(ctx); err != nil {
			fmt.Printf("fatal: commit: %v\n", err)
			return
		}
	}
}

type message struct {
	CustomerID string `json:"customer_id"`
	Type       string `json:"type"`
	Body       string `json:"body"`
}

type kafkaConsumer struct {
	kc *kafka.Consumer
}

func (kc kafkaConsumer) Close() error {
	return kc.kc.Close()
}

func (kc kafkaConsumer) Consume(ctx context.Context) (message, error) {
	for !isCancelled(ctx) {
		event := kc.kc.Poll(50)
		switch event := event.(type) {
		case *kafka.Message:
			msg := message{}
			if err := json.Unmarshal(event.Value, &msg); err != nil {
				fmt.Printf("error: consume: %v\n", err)
				continue
			}
			return msg, nil
		case kafka.PartitionEOF:
			<-time.After(time.Second)
		case kafka.Error:
			fmt.Printf("error: consume: %v\n", event.Error())
		}
	}

	return message{}, ctx.Err()
}

func (kc kafkaConsumer) Commit(_ context.Context) error {
	_, err := kc.kc.Commit()
	if err != nil {
		return err
	}
	return nil
}

func newConsumer() (kafkaConsumer, error) {

	kc, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  "kafka:29092",
		"group.id":           "consumer",
		"auto.offset.reset":  "latest",
		"enable.auto.commit": "false",
	})
	if err != nil {
		return kafkaConsumer{}, fmt.Errorf("create Kafka consumer: %w", err)
	}

	err = kc.Subscribe("messages", func(c *kafka.Consumer, e kafka.Event) error {
		fmt.Printf("rebalance: %v\n", e)
		return nil
	})
	if err != nil {
		return kafkaConsumer{}, fmt.Errorf("subscribe: %w", err)
	}

	return kafkaConsumer{
		kc: kc,
	}, nil
}

type handler interface {
	Handle(context.Context, message) error
}

func newHandler() handler {
	return consoleHandler{}
}

type consoleHandler struct{}

func (c consoleHandler) Handle(_ context.Context, msg message) error {
	fmt.Printf("message: customer_id=%q type=%q body=%q\n", msg.CustomerID, msg.Type, msg.Body)
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
