package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/ttd2089/rate-limited-consumer-poc/internal/config"
	"github.com/ttd2089/rate-limited-consumer-poc/internal/messages"
	"github.com/ttd2089/rate-limited-consumer-poc/internal/ringbuf"
)

type appConfig struct {
	BootstrapServers string `config_key:"kafka.consumer.bootstrap-servers"`
	ProduceTopic     string `config_key:"kafka.consumer.topic"`
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

	producer, err := buildProducer(cfg)
	if err != nil {
		return fmt.Errorf("build Kafka consumer: %v", err)
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer producer.Close()

		customerIDs := []string{
			"faa108f9-0815-4035-89c4-403b4f2f7948",
			"e62358f4-47bb-4a45-9db3-a1c5ad6cdab2",
			"139b70a3-60e8-47a0-9b7d-d8a369d18417",
			"432556b3-0a3b-4dbb-83fc-187115228f67",
		}
		types := []string{
			"foo",
			"bar",
			"baz",
		}

		const maxRPS = 1000
		sends := ringbuf.New[time.Time](maxRPS)

		for !isCancelled(ctx) {

			// Make message publishing "natrually" use ~90% of its rate limit. This will ensure we
			// hit the rate limit but smooth it out some instead of sending in predictable batches.
			naturalDelay := (rand.Int63n(time.Second.Nanoseconds()/maxRPS) * 9) / 10
			<-time.After(time.Duration(naturalDelay))

			customerID := customerIDs[rand.Int()%len(customerIDs)]
			type_ := types[rand.Int()%len(types)]
			body := fmt.Sprintf("[%v]: %q message for customer %q", time.Now(), type_, customerID)

			msgValue, err := json.Marshal(messages.Message{
				CustomerID: customerID,
				Type:       type_,
				Body:       body,
			})
			if err != nil {
				panic(fmt.Errorf("failed to marshal messsages.Message to JSON: %v", err))
			}

			timestamp := time.Now()

			if sends.Len() == maxRPS {
				anchor, _ := sends.Get(0)
				nextAllowedSend := anchor.Add(time.Second)
				delay := nextAllowedSend.Sub(timestamp)
				if delay > 0 {
					fmt.Printf("info: delaying %v for rate limit\n", delay)
					<-time.After(delay)
				}
			}

			err = producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic: &cfg.ProduceTopic,
				},
				Value:     msgValue,
				Timestamp: timestamp,
			}, nil)
			if err != nil {
				fmt.Printf("error: produce message: %v", err)
				continue
			}

			sends.Push(timestamp)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("error: produce message: %v\n", ev.TopicPartition.Error)
				}
			}
		}
	}()

	wg.Wait()

	return nil
}

func buildProducer(cfg appConfig) (*kafka.Producer, error) {
	kp, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.BootstrapServers,
	})
	if err != nil {
		return nil, fmt.Errorf("create Kafka producer: %w", err)
	}
	return kp, nil
}

func isCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
