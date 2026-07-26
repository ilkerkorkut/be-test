package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/entity"
	"github.com/flotaro/be-test/services/go-svc/internal/event"
	"github.com/flotaro/be-test/services/go-svc/pkg/factory"
	"go.uber.org/zap"
)

type messageHandler func(context.Context, []byte) error

type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	topics        []string
	handler       messageHandler
	logger        *zap.Logger
}

func NewPlacementConsumer(brokers string, groupID string, db database.Store, logger *zap.Logger) (*Consumer, error) {
	return newConsumer(brokers, groupID, []string{event.TopicBetsPlaced}, func(ctx context.Context, payload []byte) error {
		var placed event.BetPlaced
		if err := json.Unmarshal(payload, &placed); err != nil {
			return fmt.Errorf("decode bet placed event: %w", err)
		}
		if err := scheduleSettlement(ctx, db, placed); err != nil {
			return err
		}
		logger.Info("bet settlement scheduled", zap.String("bet_id", placed.BetID), zap.String("user_id", placed.UserID), zap.Time("settle_at", placed.PlacedAt.Add(settlementDelay)))
		return nil
	}, logger)
}

func NewSettlementConsumer(brokers string, groupID string, settlement *SettlementService, logger *zap.Logger) (*Consumer, error) {
	return newConsumer(brokers, groupID, []string{event.TopicBetsSettlementRequested}, func(ctx context.Context, payload []byte) error {
		var requested event.BetSettlementRequested
		if err := json.Unmarshal(payload, &requested); err != nil {
			return fmt.Errorf("decode settlement requested event: %w", err)
		}
		logger.Info("settlement request received", zap.String("event_id", requested.EventID), zap.String("bet_id", requested.BetID), zap.String("user_id", requested.UserID))
		return settlement.Settle(ctx, requested)
	}, logger)
}

func newConsumer(brokers string, groupID string, topics []string, handler messageHandler, logger *zap.Logger) (*Consumer, error) {
	consumerGroup, err := factory.NewKafkaConsumerGroup(
		factory.WithKafkaConsumerBrokers(splitBrokers(brokers)),
		factory.WithKafkaConsumerGroupID(groupID),
		factory.WithKafkaConsumerAutoCommit(false),
	)
	if err != nil {
		return nil, fmt.Errorf("create kafka consumer: %w", err)
	}

	return &Consumer{consumerGroup: consumerGroup, topics: topics, handler: handler, logger: logger}, nil
}

func scheduleSettlement(_ context.Context, db database.Store, placed event.BetPlaced) error {
	return db.ScheduleSettlement(entity.SettlementSchedule{
		ID:       settlementScheduleID(placed.BetID),
		BetID:    placed.BetID,
		UserID:   placed.UserID,
		SettleAt: placed.PlacedAt.Add(settlementDelay),
	})
}

func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info("kafka consumer started", zap.Strings("topics", c.topics))
	defer c.logger.Info("kafka consumer stopped", zap.Strings("topics", c.topics))
	for ctx.Err() == nil {
		if err := c.consumerGroup.Consume(ctx, c.topics, c); err != nil && ctx.Err() == nil {
			c.logger.Error("kafka consume failed", zap.Error(err))
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
			}
		}
	}

	return nil
}

func (c *Consumer) Close() error {
	return c.consumerGroup.Close()
}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		c.logger.Debug("kafka message received", zap.String("topic", message.Topic), zap.Int32("partition", message.Partition), zap.Int64("offset", message.Offset), zap.String("key", string(message.Key)))
		if err := c.handler(session.Context(), message.Value); err != nil {
			c.logger.Error("kafka message handling failed", zap.String("topic", message.Topic), zap.Int32("partition", message.Partition), zap.Int64("offset", message.Offset), zap.Error(err))
			return err
		}

		session.MarkMessage(message, "")
		session.Commit()
		c.logger.Info("kafka message processed", zap.String("topic", message.Topic), zap.Int32("partition", message.Partition), zap.Int64("offset", message.Offset), zap.String("key", string(message.Key)))
	}

	return nil
}

func splitBrokers(brokers string) []string {
	parts := strings.Split(brokers, ",")
	result := make([]string, 0, len(parts))
	for _, broker := range parts {
		if broker = strings.TrimSpace(broker); broker != "" {
			result = append(result, broker)
		}
	}

	return result
}
