package service

import (
	"context"
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/entity"
	"go.uber.org/zap"
)

const outboxBatchSize = 100

type EventPublisher interface {
	Publish(ctx context.Context, topic string, key string, payload []byte) error
}

type OutboxDispatcher struct {
	db        database.Store
	publisher EventPublisher
	logger    *zap.Logger
}

func NewOutboxDispatcher(db database.Store, publisher EventPublisher, logger *zap.Logger) *OutboxDispatcher {
	return &OutboxDispatcher{db: db, publisher: publisher, logger: logger}
}

func (d *OutboxDispatcher) Run(ctx context.Context, interval time.Duration) {
	d.logger.Info("outbox dispatcher started", zap.Duration("interval", interval))
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer d.logger.Info("outbox dispatcher stopped")

	for {
		if err := d.Dispatch(ctx); err != nil {
			d.logger.Error("failed to dispatch outbox events", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (d *OutboxDispatcher) Dispatch(ctx context.Context) error {
	events, err := d.db.GetUnpublishedOutbox(outboxBatchSize)
	if err != nil {
		return err
	}
	if len(events) > 0 {
		d.logger.Info("dispatching outbox events", zap.Int("count", len(events)))
	}

	for _, event := range events {
		if err := d.publish(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

func (d *OutboxDispatcher) publish(ctx context.Context, event entity.OutboxEvent) error {
	if err := d.publisher.Publish(ctx, event.Topic, event.Key, event.Payload); err != nil {
		d.logger.Error("outbox event publish failed", zap.String("event_id", event.ID), zap.String("topic", event.Topic), zap.String("key", event.Key), zap.Error(err))
		return err
	}
	if err := d.db.MarkOutboxPublished(event.ID); err != nil {
		return err
	}
	d.logger.Info("outbox event published", zap.String("event_id", event.ID), zap.String("topic", event.Topic), zap.String("key", event.Key))
	return nil
}
