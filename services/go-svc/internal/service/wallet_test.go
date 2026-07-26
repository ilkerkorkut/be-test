package service

import (
	"context"
	"sync"
	"testing"

	"github.com/flotaro/be-test/services/go-svc/internal/request"
)

func TestWalletServiceCreditAndDebitAreAtomic(t *testing.T) {
	t.Parallel()

	const (
		userID     = "user-1"
		operations = 250
	)

	walletService := NewWalletService(newTestStore(t))
	runParallelWalletOperation(t, operations, func() error {
		_, err := walletService.Credit(context.Background(), userID, 1, "DEPOSIT")
		return err
	})

	wallet, err := walletService.Get(context.Background(), request.GetWalletRequest{UserID: userID})
	if err != nil {
		t.Fatalf("get wallet after credits: %v", err)
	}
	if wallet.Balance != operations {
		t.Fatalf("balance after credits = %.2f, want %d", wallet.Balance, operations)
	}

	runParallelWalletOperation(t, operations, func() error {
		_, err := walletService.Debit(context.Background(), userID, 1, "WITHDRAWAL")
		return err
	})

	wallet, err = walletService.Get(context.Background(), request.GetWalletRequest{UserID: userID})
	if err != nil {
		t.Fatalf("get wallet after debits: %v", err)
	}
	if wallet.Balance != 0 {
		t.Fatalf("balance after debits = %.2f, want 0", wallet.Balance)
	}
}

func TestWalletServiceGetReturnsTenLatestLedgerEntries(t *testing.T) {
	walletService := NewWalletService(newTestStore(t))
	for index := 0; index < 12; index++ {
		if _, err := walletService.Credit(context.Background(), "user-1", 1, "DEPOSIT"); err != nil {
			t.Fatalf("credit wallet %d: %v", index, err)
		}
	}

	wallet, err := walletService.Get(context.Background(), request.GetWalletRequest{UserID: "user-1"})
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if len(wallet.Ledger) != 10 {
		t.Fatalf("ledger entries = %d, want 10", len(wallet.Ledger))
	}
}

func runParallelWalletOperation(t *testing.T, operations int, operation func() error) {
	t.Helper()

	errors := make(chan error, operations)
	var waitGroup sync.WaitGroup
	for index := 0; index < operations; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			errors <- operation()
		}()
	}
	waitGroup.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Fatalf("parallel wallet operation: %v", err)
		}
	}
}
