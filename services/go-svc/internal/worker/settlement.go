package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/entity"
	"github.com/flotaro/be-test/services/go-svc/internal/enums"
	"github.com/flotaro/be-test/services/go-svc/internal/event"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SettlementService struct {
	db     database.Store
	rand   func() float64
	logger *zap.Logger
}

func NewSettlementService(db database.Store, loggers ...*zap.Logger) *SettlementService {
	logger := zap.NewNop()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &SettlementService{db: db, rand: rand.Float64, logger: logger}
}

func (s *SettlementService) Settle(_ context.Context, requested event.BetSettlementRequested) error {
	bet, err := s.db.GetBet(requested.BetID)
	if err != nil {
		s.logger.Error("settlement bet lookup failed", zap.String("event_id", requested.EventID), zap.String("bet_id", requested.BetID), zap.Error(err))
		return err
	}

	settledAt := time.Now().UTC()
	status := enums.BetStatusLost
	var payoutCents *int64
	var ledger *entity.Ledger
	if s.rand() < 0.45 {
		status = enums.BetStatusWon
		payout := int64(math.Round(float64(bet.StakeCents) * bet.Odds))
		payoutCents = &payout
		ledger = &entity.Ledger{ID: uuid.NewString(), UserID: bet.UserID, Type: "BET_CREDIT", Amount: payout, CreatedAt: settledAt}
	}

	outboxID := uuid.NewString()
	payout := 0.0
	if payoutCents != nil {
		payout = float64(*payoutCents) / 100
	}

	payload, err := json.Marshal(event.BetSettled{EventID: outboxID, BetID: bet.ID, UserID: bet.UserID, Status: string(status), Payout: payout, SettledAt: settledAt})
	if err != nil {
		return fmt.Errorf("encode bet settled event: %w", err)
	}

	_, processed, err := s.db.SettleBet(requested.EventID, bet.ID, status, payoutCents, ledger, entity.OutboxEvent{ID: outboxID, Topic: event.TopicBetsSettled, Key: bet.UserID, Payload: payload, CreatedAt: settledAt})
	if err != nil {
		s.logger.Error("bet settlement failed", zap.String("event_id", requested.EventID), zap.String("bet_id", bet.ID), zap.Error(err))
		return err
	}
	if processed {
		s.logger.Debug("duplicate settlement event ignored", zap.String("event_id", requested.EventID), zap.String("bet_id", bet.ID))
		return nil
	}
	s.logger.Info("bet settled", zap.String("event_id", requested.EventID), zap.String("bet_id", bet.ID), zap.String("user_id", bet.UserID), zap.String("status", string(status)), zap.Float64("payout", payout))
	return nil
}
