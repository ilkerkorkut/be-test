package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/entity"
	"github.com/flotaro/be-test/services/go-svc/internal/enums"
)

var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrBetNotFound = errors.New("bet not found")

type Store interface {
	CreditWallet(userID string, amountCents int64, entry entity.Ledger) (entity.Wallet, error)
	DebitWallet(userID string, amountCents int64, entry entity.Ledger) (entity.Wallet, error)
	GetWallet(userID string, ledgerLimit int) (entity.Wallet, []entity.Ledger, error)
	CreateBet(bet entity.Bet, entry entity.Ledger, outbox entity.OutboxEvent) error
	GetBets(userID string, limit int) ([]entity.Bet, error)
	GetPendingBets() ([]entity.Bet, error)
	GetBet(betID string) (entity.Bet, error)
	SettleBet(eventID string, betID string, status enums.BetStatus, payoutCents *int64, entry *entity.Ledger, outbox entity.OutboxEvent) (entity.Bet, bool, error)
	ScheduleSettlement(schedule entity.SettlementSchedule) error
	GetDueSettlementSchedules(now time.Time, limit int) ([]entity.SettlementSchedule, error)
	DispatchSettlementSchedule(scheduleID string, now time.Time, outbox entity.OutboxEvent) (bool, error)
	QueueOutbox(event entity.OutboxEvent) error
	GetUnpublishedOutbox(limit int) ([]entity.OutboxEvent, error)
	MarkOutboxPublished(eventID string) error
	Close() error
}

type InitialWallet struct {
	UserID       string
	Currency     string
	BalanceCents int64
	CreatedAt    time.Time
}

type jsonStore struct {
	path     string
	lockPath string
}

type stateDocument struct {
	Wallets             map[string]entity.Wallet             `json:"wallets"`
	Ledgers             []entity.Ledger                      `json:"ledgers"`
	Bets                map[string]entity.Bet                `json:"bets"`
	Outbox              map[string]entity.OutboxEvent        `json:"outbox"`
	SettlementSchedules map[string]entity.SettlementSchedule `json:"settlementSchedules"`
	ProcessedEvents     map[string]time.Time                 `json:"processedEvents"`
}

func NewJSONStore(path string, initialWallets ...map[string]InitialWallet) (Store, error) {
	if path == "" {
		return nil, errors.New("database path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	store := &jsonStore{path: path, lockPath: path + ".lock"}
	if err := store.withTransaction(func(state *stateDocument) error {
		if len(initialWallets) == 0 {
			return nil
		}
		for _, seed := range initialWallets[0] {
			if _, exists := state.Wallets[seed.UserID]; exists {
				continue
			}
			state.Wallets[seed.UserID] = entity.Wallet{
				UserID:       seed.UserID,
				Currency:     seed.Currency,
				BalanceCents: seed.BalanceCents,
			}
			if seed.BalanceCents > 0 {
				state.Ledgers = append(state.Ledgers, entity.Ledger{
					ID:        "initial-balance-" + seed.UserID,
					UserID:    seed.UserID,
					Type:      "INITIAL_BALANCE",
					Amount:    seed.BalanceCents,
					CreatedAt: seed.CreatedAt,
				})
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *jsonStore) CreditWallet(userID string, amountCents int64, entry entity.Ledger) (entity.Wallet, error) {
	if amountCents <= 0 {
		return entity.Wallet{}, errors.New("credit amount must be positive")
	}

	var wallet entity.Wallet
	err := s.withTransaction(func(state *stateDocument) error {
		wallet = state.walletForUser(userID)
		wallet.BalanceCents += amountCents
		state.Wallets[userID] = wallet
		state.Ledgers = append(state.Ledgers, entry)
		return nil
	})
	return wallet, err
}

func (s *jsonStore) DebitWallet(userID string, amountCents int64, entry entity.Ledger) (entity.Wallet, error) {
	if amountCents <= 0 {
		return entity.Wallet{}, errors.New("debit amount must be positive")
	}

	var wallet entity.Wallet
	err := s.withTransaction(func(state *stateDocument) error {
		wallet = state.walletForUser(userID)
		if wallet.BalanceCents < amountCents {
			return ErrInsufficientBalance
		}
		wallet.BalanceCents -= amountCents
		state.Wallets[userID] = wallet
		state.Ledgers = append(state.Ledgers, entry)
		return nil
	})
	return wallet, err
}

func (s *jsonStore) GetWallet(userID string, ledgerLimit int) (entity.Wallet, []entity.Ledger, error) {
	var wallet entity.Wallet
	var ledger []entity.Ledger
	err := s.withTransaction(func(state *stateDocument) error {
		wallet = state.walletForUser(userID)
		ledger = state.ledgerForUser(userID, ledgerLimit)
		return nil
	})
	return wallet, ledger, err
}

func (s *jsonStore) CreateBet(bet entity.Bet, entry entity.Ledger, outbox entity.OutboxEvent) error {
	return s.withTransaction(func(state *stateDocument) error {
		wallet := state.walletForUser(bet.UserID)
		if wallet.BalanceCents < bet.StakeCents {
			return ErrInsufficientBalance
		}
		wallet.BalanceCents -= bet.StakeCents
		state.Wallets[bet.UserID] = wallet
		state.Ledgers = append(state.Ledgers, entry)
		state.Bets[bet.ID] = cloneBet(bet)
		state.Outbox[outbox.ID] = cloneOutbox(outbox)
		return nil
	})
}

func (s *jsonStore) GetBets(userID string, limit int) ([]entity.Bet, error) {
	var bets []entity.Bet
	err := s.withTransaction(func(state *stateDocument) error {
		for _, bet := range state.Bets {
			if bet.UserID == userID {
				bets = append(bets, cloneBet(bet))
			}
		}
		sort.Slice(bets, func(i, j int) bool { return bets[i].PlacedAt.After(bets[j].PlacedAt) })
		if limit > 0 && len(bets) > limit {
			bets = bets[:limit]
		}
		return nil
	})
	return bets, err
}

func (s *jsonStore) GetPendingBets() ([]entity.Bet, error) {
	bets := make([]entity.Bet, 0)
	err := s.withTransaction(func(state *stateDocument) error {
		for _, bet := range state.Bets {
			if bet.Status == enums.BetStatusPending {
				bets = append(bets, cloneBet(bet))
			}
		}
		sort.Slice(bets, func(i, j int) bool {
			if bets[i].PlacedAt.Equal(bets[j].PlacedAt) {
				return bets[i].ID < bets[j].ID
			}
			return bets[i].PlacedAt.Before(bets[j].PlacedAt)
		})
		return nil
	})
	return bets, err
}

func (s *jsonStore) GetBet(betID string) (entity.Bet, error) {
	var bet entity.Bet
	err := s.withTransaction(func(state *stateDocument) error {
		value, ok := state.Bets[betID]
		if !ok {
			return ErrBetNotFound
		}
		bet = cloneBet(value)
		return nil
	})
	return bet, err
}

func (s *jsonStore) SettleBet(eventID string, betID string, status enums.BetStatus, payoutCents *int64, entry *entity.Ledger, outbox entity.OutboxEvent) (entity.Bet, bool, error) {
	var bet entity.Bet
	processed := false
	err := s.withTransaction(func(state *stateDocument) error {
		if _, ok := state.ProcessedEvents[eventID]; ok {
			processed = true
			return nil
		}
		value, ok := state.Bets[betID]
		if !ok {
			return ErrBetNotFound
		}
		if value.Status != enums.BetStatusPending {
			state.ProcessedEvents[eventID] = time.Now().UTC()
			processed = true
			bet = cloneBet(value)
			return nil
		}

		now := time.Now().UTC()
		value.Status = status
		value.SettledAt = &now
		value.PayoutCents = cloneInt64(payoutCents)
		state.Bets[betID] = value
		if entry != nil && payoutCents != nil && *payoutCents > 0 {
			wallet := state.walletForUser(value.UserID)
			wallet.BalanceCents += *payoutCents
			state.Wallets[value.UserID] = wallet
			state.Ledgers = append(state.Ledgers, *entry)
		}
		state.Outbox[outbox.ID] = cloneOutbox(outbox)
		state.ProcessedEvents[eventID] = now
		bet = cloneBet(value)
		return nil
	})
	return bet, processed, err
}

func (s *jsonStore) QueueOutbox(event entity.OutboxEvent) error {
	if event.ID == "" || event.Topic == "" {
		return errors.New("outbox event id and topic are required")
	}

	return s.withTransaction(func(state *stateDocument) error {
		if _, exists := state.Outbox[event.ID]; exists {
			return nil
		}
		state.Outbox[event.ID] = cloneOutbox(event)
		return nil
	})
}

func (s *jsonStore) ScheduleSettlement(schedule entity.SettlementSchedule) error {
	if schedule.ID == "" || schedule.BetID == "" || schedule.UserID == "" || schedule.SettleAt.IsZero() {
		return errors.New("settlement schedule id, bet id, user id and settle time are required")
	}

	return s.withTransaction(func(state *stateDocument) error {
		if _, exists := state.SettlementSchedules[schedule.ID]; exists {
			return nil
		}
		state.SettlementSchedules[schedule.ID] = cloneSettlementSchedule(schedule)
		return nil
	})
}

func (s *jsonStore) GetDueSettlementSchedules(now time.Time, limit int) ([]entity.SettlementSchedule, error) {
	schedules := make([]entity.SettlementSchedule, 0)
	err := s.withTransaction(func(state *stateDocument) error {
		for _, schedule := range state.SettlementSchedules {
			if schedule.DispatchedAt == nil && !schedule.SettleAt.After(now) {
				schedules = append(schedules, cloneSettlementSchedule(schedule))
			}
		}
		sort.Slice(schedules, func(i, j int) bool { return schedules[i].SettleAt.Before(schedules[j].SettleAt) })
		if limit > 0 && len(schedules) > limit {
			schedules = schedules[:limit]
		}
		return nil
	})
	return schedules, err
}

func (s *jsonStore) DispatchSettlementSchedule(scheduleID string, now time.Time, outbox entity.OutboxEvent) (bool, error) {
	if scheduleID == "" || outbox.ID == "" || outbox.Topic == "" {
		return false, errors.New("settlement schedule id and outbox event id and topic are required")
	}

	dispatched := false
	err := s.withTransaction(func(state *stateDocument) error {
		schedule, exists := state.SettlementSchedules[scheduleID]
		if !exists {
			return fmt.Errorf("settlement schedule %s not found", scheduleID)
		}
		if schedule.DispatchedAt != nil || schedule.SettleAt.After(now) {
			return nil
		}

		dispatchedAt := now.UTC()
		schedule.DispatchedAt = &dispatchedAt
		state.SettlementSchedules[scheduleID] = schedule
		if _, exists := state.Outbox[outbox.ID]; !exists {
			state.Outbox[outbox.ID] = cloneOutbox(outbox)
		}
		dispatched = true
		return nil
	})
	return dispatched, err
}

func (s *jsonStore) GetUnpublishedOutbox(limit int) ([]entity.OutboxEvent, error) {
	events := make([]entity.OutboxEvent, 0)
	err := s.withTransaction(func(state *stateDocument) error {
		for _, event := range state.Outbox {
			if event.PublishedAt == nil {
				events = append(events, cloneOutbox(event))
			}
		}
		sort.Slice(events, func(i, j int) bool { return events[i].CreatedAt.Before(events[j].CreatedAt) })
		if limit > 0 && len(events) > limit {
			events = events[:limit]
		}
		return nil
	})
	return events, err
}

func (s *jsonStore) MarkOutboxPublished(eventID string) error {
	return s.withTransaction(func(state *stateDocument) error {
		event, ok := state.Outbox[eventID]
		if !ok {
			return fmt.Errorf("outbox event %s not found", eventID)
		}
		if event.PublishedAt == nil {
			now := time.Now().UTC()
			event.PublishedAt = &now
			state.Outbox[eventID] = event
		}
		return nil
	})
}

func (s *jsonStore) Close() error { return nil }

func (s *jsonStore) withTransaction(fn func(*stateDocument) error) error {
	lock, err := os.OpenFile(s.lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open database lock: %w", err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock database: %w", err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	state, err := s.readState()
	if err != nil {
		return err
	}
	if err := fn(state); err != nil {
		return err
	}
	return s.writeState(state)
}

func (s *jsonStore) readState() (*stateDocument, error) {
	content, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return newStateDocument(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read database: %w", err)
	}

	state := newStateDocument()
	if err := json.Unmarshal(content, state); err != nil {
		return nil, fmt.Errorf("decode database: %w", err)
	}
	state.initialize()
	return state, nil
}

func (s *jsonStore) writeState(state *stateDocument) error {
	content, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode database: %w", err)
	}

	temporaryFile, err := os.CreateTemp(filepath.Dir(s.path), ".go-svc-*.json")
	if err != nil {
		return fmt.Errorf("create temporary database: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporaryFile.Write(content); err != nil {
		temporaryFile.Close()
		return fmt.Errorf("write temporary database: %w", err)
	}
	if err := temporaryFile.Sync(); err != nil {
		temporaryFile.Close()
		return fmt.Errorf("sync temporary database: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf("close temporary database: %w", err)
	}
	if err := os.Rename(temporaryPath, s.path); err != nil {
		return fmt.Errorf("replace database: %w", err)
	}
	return nil
}

func newStateDocument() *stateDocument {
	return &stateDocument{Wallets: map[string]entity.Wallet{}, Ledgers: []entity.Ledger{}, Bets: map[string]entity.Bet{}, Outbox: map[string]entity.OutboxEvent{}, SettlementSchedules: map[string]entity.SettlementSchedule{}, ProcessedEvents: map[string]time.Time{}}
}

func (s *stateDocument) initialize() {
	if s.Wallets == nil {
		s.Wallets = map[string]entity.Wallet{}
	}
	if s.Ledgers == nil {
		s.Ledgers = []entity.Ledger{}
	}
	if s.Bets == nil {
		s.Bets = map[string]entity.Bet{}
	}
	if s.Outbox == nil {
		s.Outbox = map[string]entity.OutboxEvent{}
	}
	if s.SettlementSchedules == nil {
		s.SettlementSchedules = map[string]entity.SettlementSchedule{}
	}
	if s.ProcessedEvents == nil {
		s.ProcessedEvents = map[string]time.Time{}
	}
}

func (s *stateDocument) walletForUser(userID string) entity.Wallet {
	wallet, ok := s.Wallets[userID]
	if !ok {
		return entity.Wallet{UserID: userID, Currency: "USD"}
	}
	return wallet
}

func (s *stateDocument) ledgerForUser(userID string, limit int) []entity.Ledger {
	ledger := make([]entity.Ledger, 0)
	for _, entry := range s.Ledgers {
		if entry.UserID == userID {
			ledger = append(ledger, entry)
		}
	}
	sort.Slice(ledger, func(i, j int) bool { return ledger[i].CreatedAt.After(ledger[j].CreatedAt) })
	if limit > 0 && len(ledger) > limit {
		ledger = ledger[:limit]
	}
	return ledger
}

func cloneBet(bet entity.Bet) entity.Bet {
	copyBet := bet
	copyBet.PayoutCents = cloneInt64(bet.PayoutCents)
	if bet.SettledAt != nil {
		value := *bet.SettledAt
		copyBet.SettledAt = &value
	}
	return copyBet
}
func cloneOutbox(event entity.OutboxEvent) entity.OutboxEvent {
	copyEvent := event
	copyEvent.Payload = append([]byte(nil), event.Payload...)
	if event.PublishedAt != nil {
		value := *event.PublishedAt
		copyEvent.PublishedAt = &value
	}
	return copyEvent
}
func cloneSettlementSchedule(schedule entity.SettlementSchedule) entity.SettlementSchedule {
	copySchedule := schedule
	if schedule.DispatchedAt != nil {
		value := *schedule.DispatchedAt
		copySchedule.DispatchedAt = &value
	}
	return copySchedule
}
func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
