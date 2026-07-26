package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/entity"
	"github.com/flotaro/be-test/services/go-svc/internal/request"
	"github.com/flotaro/be-test/services/go-svc/internal/response"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const walletLedgerLimit = 10

var ErrInvalidAmount = errors.New("amount must be a positive value with at most two decimal places")

type WalletService interface {
	Get(ctx context.Context, req request.GetWalletRequest) (*response.WalletResponse, error)
	Credit(ctx context.Context, userID string, amount float64, entryType string) (*response.WalletResponse, error)
	Debit(ctx context.Context, userID string, amount float64, entryType string) (*response.WalletResponse, error)
}

type walletService struct {
	db     database.Store
	logger *zap.Logger
}

func NewWalletService(db database.Store, loggers ...*zap.Logger) WalletService {
	return &walletService{db: db, logger: serviceLogger(loggers)}
}

func (s *walletService) Get(_ context.Context, req request.GetWalletRequest) (*response.WalletResponse, error) {
	if err := validateUserID(req.UserID); err != nil {
		return nil, err
	}

	wallet, ledger, err := s.db.GetWallet(req.UserID, walletLedgerLimit)
	if err != nil {
		s.logger.Error("get wallet failed", zap.String("user_id", req.UserID), zap.Error(err))
		return nil, err
	}
	s.logger.Debug("wallet retrieved", zap.String("user_id", req.UserID), zap.Int("ledger_entries", len(ledger)))

	return walletResponse(wallet, ledger), nil
}

func (s *walletService) Credit(_ context.Context, userID string, amount float64, entryType string) (*response.WalletResponse, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	cents, err := toCents(amount)
	if err != nil {
		return nil, err
	}

	wallet, err := s.db.CreditWallet(userID, cents, newLedger(userID, entryType, cents))
	if err != nil {
		s.logger.Warn("wallet credit rejected", zap.String("user_id", userID), zap.Int64("amount_cents", cents), zap.String("entry_type", entryType), zap.Error(err))
		return nil, err
	}
	s.logger.Info("wallet credited", zap.String("user_id", userID), zap.Int64("amount_cents", cents), zap.String("entry_type", entryType), zap.Int64("balance_cents", wallet.BalanceCents))

	return walletResponse(wallet, nil), nil
}

func (s *walletService) Debit(_ context.Context, userID string, amount float64, entryType string) (*response.WalletResponse, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	cents, err := toCents(amount)
	if err != nil {
		return nil, err
	}

	wallet, err := s.db.DebitWallet(userID, cents, newLedger(userID, entryType, -cents))
	if err != nil {
		s.logger.Warn("wallet debit rejected", zap.String("user_id", userID), zap.Int64("amount_cents", cents), zap.String("entry_type", entryType), zap.Error(err))
		return nil, err
	}
	s.logger.Info("wallet debited", zap.String("user_id", userID), zap.Int64("amount_cents", cents), zap.String("entry_type", entryType), zap.Int64("balance_cents", wallet.BalanceCents))

	return walletResponse(wallet, nil), nil
}

func toCents(amount float64) (int64, error) {
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, ErrInvalidAmount
	}

	cents := math.Round(amount * 100)
	if math.Abs(amount*100-cents) > 0.000001 {
		return 0, ErrInvalidAmount
	}

	return int64(cents), nil
}

func newLedger(userID string, entryType string, amountCents int64) entity.Ledger {
	return entity.Ledger{
		ID:        uuid.NewString(),
		UserID:    userID,
		Type:      entryType,
		Amount:    amountCents,
		CreatedAt: time.Now().UTC(),
	}
}

func walletResponse(wallet entity.Wallet, entries []entity.Ledger) *response.WalletResponse {
	ledger := make([]response.LedgerResponse, 0, len(entries))
	for _, entry := range entries {
		ledger = append(ledger, response.LedgerResponse{
			ID:        entry.ID,
			Type:      entry.Type,
			Amount:    float64(entry.Amount) / 100,
			CreatedAt: entry.CreatedAt.Format(time.RFC3339),
		})
	}

	return &response.WalletResponse{
		Balance:  float64(wallet.BalanceCents) / 100,
		Currency: wallet.Currency,
		Ledger:   ledger,
	}
}

func validateUserID(userID string) error {
	if userID == "" {
		return fmt.Errorf("user id is required")
	}

	return nil
}
