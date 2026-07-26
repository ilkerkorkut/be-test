package database

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/entity"
	"github.com/flotaro/be-test/services/go-svc/internal/enums"
)

func TestCreateBetDebitsWalletAndQueuesOutboxAtomically(t *testing.T) {
	store, err := NewJSONStore(filepath.Join(t.TempDir(), "go-svc.json"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	if _, err := store.CreditWallet("user-1", 1000, entity.Ledger{ID: "deposit", UserID: "user-1", Type: "DEPOSIT", Amount: 1000, CreatedAt: now}); err != nil {
		t.Fatalf("credit wallet: %v", err)
	}
	payload, err := json.Marshal(map[string]string{"betId": "bet-1"})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	if err := store.CreateBet(
		entity.Bet{ID: "bet-1", UserID: "user-1", StakeCents: 400, Odds: 2, Status: enums.BetStatusPending, PlacedAt: now},
		entity.Ledger{ID: "debit", UserID: "user-1", Type: "BET_DEBIT", Amount: -400, CreatedAt: now},
		entity.OutboxEvent{ID: "event-1", Topic: "bets.placed.v1", Key: "bet-1", Payload: payload, CreatedAt: now},
	); err != nil {
		t.Fatalf("create bet: %v", err)
	}

	wallet, ledger, err := store.GetWallet("user-1", 10)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.BalanceCents != 600 || len(ledger) != 2 {
		t.Fatalf("unexpected wallet state: balance=%d ledger=%d", wallet.BalanceCents, len(ledger))
	}
	outbox, err := store.GetUnpublishedOutbox(10)
	if err != nil {
		t.Fatalf("get outbox: %v", err)
	}
	if len(outbox) != 1 || outbox[0].ID != "event-1" {
		t.Fatalf("unexpected outbox events: %+v", outbox)
	}
}
