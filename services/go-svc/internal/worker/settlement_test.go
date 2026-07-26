package worker

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/entity"
	"github.com/flotaro/be-test/services/go-svc/internal/enums"
	"github.com/flotaro/be-test/services/go-svc/internal/event"
)

func TestSettlementIsIdempotentForDuplicateKafkaEvents(t *testing.T) {
	store, err := database.NewJSONStore(filepath.Join(t.TempDir(), "go-svc.json"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	if _, err := store.CreditWallet("user-1", 1000, entity.Ledger{ID: "deposit", UserID: "user-1", Type: "DEPOSIT", Amount: 1000, CreatedAt: now}); err != nil {
		t.Fatalf("credit wallet: %v", err)
	}
	payload, _ := json.Marshal(event.BetPlaced{EventID: "placed-1", BetID: "bet-1", UserID: "user-1", PlacedAt: now})
	if err := store.CreateBet(
		entity.Bet{ID: "bet-1", UserID: "user-1", StakeCents: 500, Odds: 2, Status: enums.BetStatusPending, PlacedAt: now},
		entity.Ledger{ID: "debit", UserID: "user-1", Type: "BET_DEBIT", Amount: -500, CreatedAt: now},
		entity.OutboxEvent{ID: "placed-1", Topic: event.TopicBetsPlaced, Key: "bet-1", Payload: payload, CreatedAt: now},
	); err != nil {
		t.Fatalf("create bet: %v", err)
	}

	settlement := NewSettlementService(store)
	settlement.rand = func() float64 { return 0.1 }
	requested := event.BetSettlementRequested{EventID: "settlement-requested-placed-1", BetID: "bet-1", UserID: "user-1", RequestedAt: now}
	if err := settlement.Settle(t.Context(), requested); err != nil {
		t.Fatalf("settle bet: %v", err)
	}
	if err := settlement.Settle(t.Context(), requested); err != nil {
		t.Fatalf("settle duplicate bet: %v", err)
	}

	wallet, _, err := store.GetWallet("user-1", 10)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.BalanceCents != 1500 {
		t.Fatalf("wallet balance = %d, want 1500", wallet.BalanceCents)
	}
	bets, err := store.GetBets("user-1", 10)
	if err != nil {
		t.Fatalf("get bets: %v", err)
	}
	if len(bets) != 1 || bets[0].Status != enums.BetStatusWon {
		t.Fatalf("unexpected bet state: %+v", bets)
	}
}
