package factory

import (
	"time"

	"github.com/IBM/sarama"
)

type KafkaConsumerConfig struct {
	Brokers        []string
	ClientID       string
	GroupID        string
	ReturnErrors   bool
	IsolationLevel sarama.IsolationLevel
	RetryBackoff   time.Duration
	SessionTimeout time.Duration
	AutoCommit     bool
	InitialOffset  int64
}

type KafkaConsumerOption func(*KafkaConsumerConfig)

func WithKafkaConsumerBrokers(brokers []string) KafkaConsumerOption {
	return func(c *KafkaConsumerConfig) {
		c.Brokers = brokers
	}
}

func WithKafkaConsumerGroupID(groupID string) KafkaConsumerOption {
	return func(c *KafkaConsumerConfig) {
		c.GroupID = groupID
	}
}

func WithKafkaConsumerReturnErrors(returnErrors bool) KafkaConsumerOption {
	return func(c *KafkaConsumerConfig) {
		c.ReturnErrors = returnErrors
	}
}

func WithKafkaConsumerIsolationLevel(isolationLevel sarama.IsolationLevel) KafkaConsumerOption {
	return func(c *KafkaConsumerConfig) {
		c.IsolationLevel = isolationLevel
	}
}

func WithKafkaConsumerRetryBackoff(retryBackoff time.Duration) KafkaConsumerOption {
	return func(c *KafkaConsumerConfig) {
		c.RetryBackoff = retryBackoff
	}
}

func WithKafkaConsumerSessionTimeout(sessionTimeout time.Duration) KafkaConsumerOption {
	return func(c *KafkaConsumerConfig) {
		c.SessionTimeout = sessionTimeout
	}
}

func WithKafkaConsumerAutoCommit(autoCommit bool) KafkaConsumerOption {
	return func(c *KafkaConsumerConfig) {
		c.AutoCommit = autoCommit
	}
}

func WithClientID(clientID string) KafkaConsumerOption {
	return func(c *KafkaConsumerConfig) {
		c.ClientID = clientID
	}
}

//nolint:mnd
func NewKafkaConsumerGroup(
	opts ...KafkaConsumerOption,
) (sarama.ConsumerGroup, error) {
	c := &KafkaConsumerConfig{
		Brokers:        []string{"localhost:39092"},
		ReturnErrors:   false,
		IsolationLevel: sarama.ReadUncommitted,
		RetryBackoff:   2 * time.Second,
		SessionTimeout: 10 * time.Second,
		AutoCommit:     true,
		InitialOffset:  sarama.OffsetOldest,
	}

	for _, opt := range opts {
		opt(c)
	}

	config := sarama.NewConfig()
	config.ClientID = c.ClientID
	config.Consumer.Return.Errors = c.ReturnErrors
	config.Consumer.IsolationLevel = c.IsolationLevel
	config.Consumer.Retry.Backoff = c.RetryBackoff
	config.Consumer.Group.Session.Timeout = c.SessionTimeout
	config.Consumer.Offsets.AutoCommit.Enable = c.AutoCommit
	config.Consumer.Offsets.Initial = c.InitialOffset

	client, err := sarama.NewConsumerGroup(c.Brokers, c.GroupID, config)
	if err != nil {
		return nil, err
	}

	return client, nil
}
