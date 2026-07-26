package factory

import (
	"time"

	"github.com/IBM/sarama"
)

type KafkaTransactionConfig struct {
	ID      string
	Timeout time.Duration
	Retry   struct {
		Max         int
		Backoff     time.Duration
		BackoffFunc func(retries, maxRetries int) time.Duration
	}
}

type KafkaProducerConfig struct {
	Brokers         []string
	RequiredAcks    sarama.RequiredAcks
	RetryMax        int
	RetryBackoff    time.Duration
	ReturnErrors    bool
	ReturnSuccesses bool
	MaxMessageBytes int
	Timeout         time.Duration
	Idempotent      bool
	Interceptors    []sarama.ProducerInterceptor
	Transaction     *KafkaTransactionConfig
}

type KafkaProducerOption func(*KafkaProducerConfig)

func WithKafkaProducerBrokers(brokers []string) KafkaProducerOption {
	return func(c *KafkaProducerConfig) {
		c.Brokers = brokers
	}
}

func WithKafkaProducerRequiredAcks(requiredAcks sarama.RequiredAcks) KafkaProducerOption {
	return func(c *KafkaProducerConfig) {
		c.RequiredAcks = requiredAcks
	}
}

func WithKafkaProducerRetryMax(retryMax int) KafkaProducerOption {
	return func(c *KafkaProducerConfig) {
		c.RetryMax = retryMax
	}
}

func WithKafkaProducerRetryBackoff(retryBackoff time.Duration) KafkaProducerOption {
	return func(c *KafkaProducerConfig) {
		c.RetryBackoff = retryBackoff
	}
}

func WithKafkaProducerReturnErrors(returnErrors bool) KafkaProducerOption {
	return func(c *KafkaProducerConfig) {
		c.ReturnErrors = returnErrors
	}
}

func WithKafkaProducerReturnSuccesses(returnSuccesses bool) KafkaProducerOption {
	return func(c *KafkaProducerConfig) {
		c.ReturnSuccesses = returnSuccesses
	}
}

func WithKafkaProducerMaxMessageBytes(maxMessageBytes int) KafkaProducerOption {
	return func(c *KafkaProducerConfig) {
		c.MaxMessageBytes = maxMessageBytes
	}
}

func WithKafkaProducerTimeout(timeout time.Duration) KafkaProducerOption {
	return func(c *KafkaProducerConfig) {
		c.Timeout = timeout
	}
}

func WithKafkaProducerIdempotent(idempotent bool) KafkaProducerOption {
	return func(c *KafkaProducerConfig) {
		c.Idempotent = idempotent
	}
}

func WithKafkaProducerInterceptors(interceptors []sarama.ProducerInterceptor) KafkaProducerOption {
	return func(c *KafkaProducerConfig) {
		c.Interceptors = interceptors
	}
}

func WithKafkaProducerTransaction(transaction *KafkaTransactionConfig) KafkaProducerOption {
	return func(c *KafkaProducerConfig) {
		c.Transaction = transaction
	}
}

//nolint:mnd
func NewKafkaSyncProducer(opts ...KafkaProducerOption) (sarama.SyncProducer, error) {
	cfg := KafkaProducerConfig{
		Brokers:         []string{"localhost:39092"},
		RequiredAcks:    sarama.WaitForAll,
		RetryMax:        5,
		RetryBackoff:    100 * time.Millisecond,
		ReturnErrors:    true,
		ReturnSuccesses: true,
		MaxMessageBytes: 1000000,
		Timeout:         10 * time.Second,
		Idempotent:      true,
		Interceptors:    nil,
		Transaction:     nil,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	config := sarama.NewConfig()

	config.Producer.RequiredAcks = cfg.RequiredAcks
	config.Producer.Retry.Max = cfg.RetryMax
	config.Producer.Return.Successes = cfg.ReturnSuccesses
	config.Producer.Return.Errors = cfg.ReturnErrors
	config.Producer.MaxMessageBytes = cfg.MaxMessageBytes
	config.Producer.Timeout = cfg.Timeout
	config.Producer.Idempotent = cfg.Idempotent
	config.Producer.Interceptors = cfg.Interceptors

	if cfg.Idempotent {
		config.Net.MaxOpenRequests = 1
	}

	if cfg.Transaction != nil {
		config.Producer.Transaction.ID = cfg.Transaction.ID
		config.Producer.Transaction.Timeout = cfg.Transaction.Timeout
		config.Producer.Transaction.Retry.Max = cfg.Transaction.Retry.Max
		config.Producer.Transaction.Retry.Backoff = cfg.Transaction.Retry.Backoff
		config.Producer.Transaction.Retry.BackoffFunc = cfg.Transaction.Retry.BackoffFunc
	}

	producer, err := sarama.NewSyncProducer(cfg.Brokers, config)
	if err != nil {
		return nil, err
	}

	return producer, nil
}

//nolint:mnd
func NewKafkaAsyncProducer(opts ...KafkaProducerOption) (sarama.AsyncProducer, error) {
	cfg := KafkaProducerConfig{
		Brokers:         []string{"localhost:39092"},
		RequiredAcks:    sarama.WaitForAll,
		RetryMax:        5,
		RetryBackoff:    100 * time.Millisecond,
		ReturnErrors:    true,
		ReturnSuccesses: true,
		MaxMessageBytes: 1000000,
		Timeout:         10 * time.Second,
		Idempotent:      true,
		Interceptors:    nil,
		Transaction:     nil,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	config := sarama.NewConfig()

	config.Producer.RequiredAcks = cfg.RequiredAcks
	config.Producer.Retry.Max = cfg.RetryMax
	config.Producer.Return.Successes = cfg.ReturnSuccesses
	config.Producer.Return.Errors = cfg.ReturnErrors
	config.Producer.MaxMessageBytes = cfg.MaxMessageBytes
	config.Producer.Timeout = cfg.Timeout
	config.Producer.Idempotent = cfg.Idempotent
	config.Producer.Interceptors = cfg.Interceptors

	if cfg.Transaction != nil {
		config.Producer.Transaction.ID = cfg.Transaction.ID
		config.Producer.Transaction.Timeout = cfg.Transaction.Timeout
		config.Producer.Transaction.Retry.Max = cfg.Transaction.Retry.Max
		config.Producer.Transaction.Retry.Backoff = cfg.Transaction.Retry.Backoff
		config.Producer.Transaction.Retry.BackoffFunc = cfg.Transaction.Retry.BackoffFunc
	}

	producer, err := sarama.NewAsyncProducer(cfg.Brokers, config)
	if err != nil {
		return nil, err
	}

	return producer, nil
}
