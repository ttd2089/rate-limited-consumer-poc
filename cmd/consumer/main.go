package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/ttd2089/rate-limited-consumer-poc/internal/config"
	"github.com/ttd2089/rate-limited-consumer-poc/internal/messages"
)

type appConfig struct {
	BootstrapServers string `config_key:"kafka.consumer.bootstrap-servers"`
	ConsumerGroupID  string `config_key:"kafka.consumer.group-id"`
	ConsumeTopic     string `config_key:"kafka.consumer.topic"`
}

func main() {
	if err := run(); err != nil {
		fmt.Printf("fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg, err := config.Parse[appConfig](config.EnvMap{})
	if err != nil {
		return fmt.Errorf("parse app config: %v", err)
	}

	consumer, err := buildConsumer(cfg)
	if err != nil {
		return fmt.Errorf("build Kafka consumer: %v", err)
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
			return fmt.Errorf("handle msg: %v", err)
		}

		if err := consumer.Commit(ctx); err != nil {
			return fmt.Errorf("commit: %v\n", err)
		}
	}

	return nil
}

type kafkaConsumer struct {
	kc *kafka.Consumer
}

func (kc kafkaConsumer) Close() error {
	return kc.kc.Close()
}

func (kc kafkaConsumer) Consume(ctx context.Context) (messages.Message, error) {
	for !isCancelled(ctx) {
		event := kc.kc.Poll(50)
		switch event := event.(type) {
		case *kafka.Message:
			msg := messages.Message{}
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

	return messages.Message{}, ctx.Err()
}

func (kc kafkaConsumer) Commit(_ context.Context) error {
	_, err := kc.kc.Commit()
	if err != nil {
		return err
	}
	return nil
}

func buildConsumer(cfg appConfig) (kafkaConsumer, error) {

	kc, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  cfg.BootstrapServers,
		"group.id":           cfg.ConsumerGroupID,
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
	Handle(context.Context, messages.Message) error
}

func newHandler() handler {
	return consoleHandler{}
}

type consoleHandler struct{}

func (c consoleHandler) Handle(_ context.Context, msg messages.Message) error {
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
