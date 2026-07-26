package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/flotaro/be-test/services/go-svc/pkg/factory"
)

type Producer struct {
	producer sarama.SyncProducer
}

const (
	producerStartupAttempts = 15
	producerStartupBackoff  = time.Second
)

func NewProducer(brokers string) (*Producer, error) {
	var lastErr error
	for attempt := 1; attempt <= producerStartupAttempts; attempt++ {
		producer, err := factory.NewKafkaSyncProducer(factory.WithKafkaProducerBrokers(splitBrokers(brokers)))
		if err == nil {
			return &Producer{producer: producer}, nil
		}
		lastErr = err
		if attempt < producerStartupAttempts {
			time.Sleep(producerStartupBackoff)
		}
	}

	return nil, fmt.Errorf("create kafka producer after %d attempts: %w", producerStartupAttempts, lastErr)
}

func (p *Producer) Publish(_ context.Context, topic string, key string, payload []byte) error {
	_, _, err := p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(payload),
	})
	if err != nil {
		return fmt.Errorf("publish kafka message: %w", err)
	}

	return nil
}

func (p *Producer) Close() error {
	return p.producer.Close()
}

func splitBrokers(brokers string) []string {
	values := strings.Split(brokers, ",")
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}

	return result
}
