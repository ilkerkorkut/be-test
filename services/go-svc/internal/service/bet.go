package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/entity"
	"github.com/flotaro/be-test/services/go-svc/internal/enums"
	"github.com/flotaro/be-test/services/go-svc/internal/event"
	"github.com/flotaro/be-test/services/go-svc/internal/request"
	"github.com/flotaro/be-test/services/go-svc/internal/response"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type BetService interface {
	Place(ctx context.Context, req request.PlaceBetRequest) (*response.BetResponse, error)
	Get(ctx context.Context, req request.GetBetsRequest) ([]response.BetHistoryResponse, error)
}

type betService struct {
	db     database.Store
	logger *zap.Logger
}

func NewBetService(db database.Store, loggers ...*zap.Logger) BetService {
	return &betService{db: db, logger: serviceLogger(loggers)}
}

func (s *betService) Place(_ context.Context, req request.PlaceBetRequest) (*response.BetResponse, error) {
	if err := validateBetRequest(req); err != nil {
		return nil, err
	}

	stakeCents, err := toCents(req.Stake)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	bet := entity.Bet{
		ID:         uuid.NewString(),
		UserID:     req.UserID,
		EventID:    req.EventID,
		SportKey:   req.SportKey,
		Market:     req.Market,
		Selection:  req.Selection,
		StakeCents: stakeCents,
		Odds:       req.Odds,
		Status:     enums.BetStatusPending,
		PlacedAt:   now,
	}
	outboxID := uuid.NewString()
	payload, err := json.Marshal(event.BetPlaced{
		EventID:  outboxID,
		BetID:    bet.ID,
		UserID:   bet.UserID,
		PlacedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("encode bet placed event: %w", err)
	}

	if err := s.db.CreateBet(bet, newLedger(bet.UserID, "BET_DEBIT", -stakeCents), entity.OutboxEvent{
		ID:        outboxID,
		Topic:     event.TopicBetsPlaced,
		Key:       bet.ID,
		Payload:   payload,
		CreatedAt: now,
	}); err != nil {
		s.logger.Warn("bet placement rejected", zap.String("user_id", req.UserID), zap.String("event_id", req.EventID), zap.Int64("stake_cents", stakeCents), zap.Error(err))
		return nil, err
	}
	s.logger.Info("bet placed", zap.String("bet_id", bet.ID), zap.String("user_id", bet.UserID), zap.String("event_id", bet.EventID), zap.Int64("stake_cents", bet.StakeCents), zap.Float64("odds", bet.Odds))

	return &response.BetResponse{BetID: bet.ID, Status: bet.Status}, nil
}

func (s *betService) Get(_ context.Context, req request.GetBetsRequest) ([]response.BetHistoryResponse, error) {
	if err := validateUserID(req.UserID); err != nil {
		return nil, err
	}
	if req.Limit < 1 {
		return nil, errors.New("limit must be positive")
	}

	bets, err := s.db.GetBets(req.UserID, req.Limit)
	if err != nil {
		s.logger.Error("get bet history failed", zap.String("user_id", req.UserID), zap.Error(err))
		return nil, err
	}
	s.logger.Debug("bet history retrieved", zap.String("user_id", req.UserID), zap.Int("count", len(bets)))
	responses := make([]response.BetHistoryResponse, 0, len(bets))
	for _, bet := range bets {
		responses = append(responses, betResponse(bet))
	}

	return responses, nil
}

func serviceLogger(loggers []*zap.Logger) *zap.Logger {
	if len(loggers) > 0 && loggers[0] != nil {
		return loggers[0]
	}
	return zap.NewNop()
}

func validateBetRequest(req request.PlaceBetRequest) error {
	if req.UserID == "" || req.EventID == "" || req.SportKey == "" || req.Market == "" || req.Selection == "" {
		return errors.New("user id, event id, sport key, market and selection are required")
	}
	if _, err := toCents(req.Stake); err != nil {
		return err
	}
	if req.Odds <= 1 || math.IsNaN(req.Odds) || math.IsInf(req.Odds, 0) {
		return fmt.Errorf("odds must be greater than 1")
	}

	return nil
}

func betResponse(bet entity.Bet) response.BetHistoryResponse {
	var payout *float64
	if bet.PayoutCents != nil {
		value := float64(*bet.PayoutCents) / 100
		payout = &value
	}
	var settledAt *string
	if bet.SettledAt != nil {
		value := bet.SettledAt.Format(time.RFC3339)
		settledAt = &value
	}

	return response.BetHistoryResponse{
		ID:        bet.ID,
		EventID:   bet.EventID,
		Market:    bet.Market,
		Selection: bet.Selection,
		Stake:     float64(bet.StakeCents) / 100,
		Odds:      bet.Odds,
		Status:    bet.Status,
		Payout:    payout,
		PlacedAt:  bet.PlacedAt.Format(time.RFC3339),
		SettledAt: settledAt,
	}
}
