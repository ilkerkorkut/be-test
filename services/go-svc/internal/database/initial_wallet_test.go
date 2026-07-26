package database

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewJSONStoreSeedsInitialWalletOnlyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go-svc.json")
	createdAt := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	seed := map[string]InitialWallet{
		"user-1": {
			UserID:       "user-1",
			Currency:     "CAD",
			BalanceCents: 423020,
			CreatedAt:    createdAt,
		},
	}

	store, err := NewJSONStore(path, seed)
	if err != nil {
		t.Fatalf("create seeded store: %v", err)
	}
	wallet, ledger, err := store.GetWallet("user-1", 10)
	if err != nil {
		t.Fatalf("get seeded wallet: %v", err)
	}
	if wallet.BalanceCents != 423020 || wallet.Currency != "CAD" {
		t.Fatalf("seeded wallet = %+v", wallet)
	}
	if len(ledger) != 1 || ledger[0].Type != "INITIAL_BALANCE" || ledger[0].Amount != 423020 {
		t.Fatalf("seeded ledger = %+v", ledger)
	}
	_ = store.Close()

	store, err = NewJSONStore(path, map[string]InitialWallet{
		"user-1": {UserID: "user-1", Currency: "USD", BalanceCents: 1, CreatedAt: createdAt.Add(time.Hour)},
	})
	if err != nil {
		t.Fatalf("reopen seeded store: %v", err)
	}
	defer store.Close()
	wallet, ledger, err = store.GetWallet("user-1", 10)
	if err != nil {
		t.Fatalf("get reopened wallet: %v", err)
	}
	if wallet.BalanceCents != 423020 || wallet.Currency != "CAD" || len(ledger) != 1 {
		t.Fatalf("wallet was reseeded: wallet=%+v ledger=%+v", wallet, ledger)
	}
}
