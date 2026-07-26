package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/entity"
	"github.com/flotaro/be-test/services/go-svc/internal/event"
	"go.uber.org/zap"
)

const (
	settlementDelay             = 30 * time.Second
	settlementScheduleBatchSize = 100
)

type SettlementScheduler struct {
	db     database.Store
	logger *zap.Logger
}

func NewSettlementScheduler(db database.Store, logger *zap.Logger) *SettlementScheduler {
	return &SettlementScheduler{db: db, logger: logger}
}

func (s *SettlementScheduler) Run(ctx context.Context, interval time.Duration) {
	s.logger.Info("settlement scheduler started", zap.Duration("interval", interval), zap.Duration("settlement_delay", settlementDelay))
	if err := s.Reconcile(); err != nil {
		s.logger.Error("failed to reconcile pending settlement schedules", zap.Error(err))
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer s.logger.Info("settlement scheduler stopped")

	for {
		if err := s.Dispatch(time.Now().UTC()); err != nil {
			s.logger.Error("failed to dispatch due settlement schedules", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *SettlementScheduler) Reconcile() error {
	bets, err := s.db.GetPendingBets()
	if err != nil {
		return err
	}
	if len(bets) > 0 {
		s.logger.Info("reconciling pending settlement schedules", zap.Int("count", len(bets)))
	}

	for _, bet := range bets {
		if err := s.db.ScheduleSettlement(entity.SettlementSchedule{
			ID:       settlementScheduleID(bet.ID),
			BetID:    bet.ID,
			UserID:   bet.UserID,
			SettleAt: bet.PlacedAt.Add(settlementDelay),
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *SettlementScheduler) Dispatch(now time.Time) error {
	schedules, err := s.db.GetDueSettlementSchedules(now, settlementScheduleBatchSize)
	if err != nil {
		return err
	}

	for _, schedule := range schedules {
		payload, err := json.Marshal(event.BetSettlementRequested{
			EventID:     schedule.ID,
			BetID:       schedule.BetID,
			UserID:      schedule.UserID,
			RequestedAt: now,
		})
		if err != nil {
			return fmt.Errorf("encode settlement requested event: %w", err)
		}

		dispatched, err := s.db.DispatchSettlementSchedule(schedule.ID, now, entity.OutboxEvent{
			ID:        schedule.ID,
			Topic:     event.TopicBetsSettlementRequested,
			Key:       schedule.BetID,
			Payload:   payload,
			CreatedAt: now,
		})
		if err != nil {
			return err
		}
		if dispatched {
			s.logger.Info("settlement request queued", zap.String("event_id", schedule.ID), zap.String("bet_id", schedule.BetID), zap.String("user_id", schedule.UserID), zap.Time("settle_at", schedule.SettleAt))
		}
	}

	return nil
}

func settlementScheduleID(betID string) string {
	return "settlement-requested-" + betID
}
