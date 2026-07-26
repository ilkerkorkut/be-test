package service

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/request"
	"github.com/flotaro/be-test/services/go-svc/internal/response"
)

func TestBetServicePlacePreventsOverspendUnderParallelRequests(t *testing.T) {
	t.Parallel()

	const (
		userID          = "user-1"
		startingBalance = 100.00
		parallelBets    = 200
	)

	db := newTestStore(t)
	walletService := NewWalletService(db)
	if _, err := walletService.Credit(context.Background(), userID, startingBalance, "DEPOSIT"); err != nil {
		t.Fatalf("credit wallet: %v", err)
	}

	betService := NewBetService(db, fixedOddsLookup{price: 2})
	results := make(chan error, parallelBets)
	var waitGroup sync.WaitGroup

	for index := 0; index < parallelBets; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			_, err := betService.Place(context.Background(), request.PlaceBetRequest{
				UserID:    userID,
				EventID:   fmt.Sprintf("event-%d", index),
				SportKey:  "soccer_epl",
				Market:    "h2h",
				Selection: "Home",
				Stake:     1,
				Odds:      2,
			})
			results <- err
		}(index)
	}

	waitGroup.Wait()
	close(results)

	successfulBets := 0
	for err := range results {
		if err == nil {
			successfulBets++
			continue
		}
		if err != database.ErrInsufficientBalance {
			t.Fatalf("unexpected place bet error: %v", err)
		}
	}

	if successfulBets != int(startingBalance) {
		t.Fatalf("successful bets = %d, want %d", successfulBets, int(startingBalance))
	}

	wallet, err := walletService.Get(context.Background(), request.GetWalletRequest{UserID: userID})
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.Balance != 0 {
		t.Fatalf("wallet balance = %.2f, want 0", wallet.Balance)
	}

	bets, err := betService.Get(context.Background(), request.GetBetsRequest{UserID: userID, Limit: parallelBets})
	if err != nil {
		t.Fatalf("get bets: %v", err)
	}
	if len(bets) != successfulBets {
		t.Fatalf("stored bets = %d, want %d", len(bets), successfulBets)
	}
}

func TestBetServiceGetRespectsLimit(t *testing.T) {
	db := newTestStore(t)
	walletService := NewWalletService(db)
	if _, err := walletService.Credit(context.Background(), "user-1", 10, "DEPOSIT"); err != nil {
		t.Fatalf("credit wallet: %v", err)
	}

	betService := NewBetService(db, fixedOddsLookup{price: 2})
	for index := 0; index < 3; index++ {
		if _, err := betService.Place(context.Background(), request.PlaceBetRequest{
			UserID:    "user-1",
			EventID:   fmt.Sprintf("event-%d", index),
			SportKey:  "soccer_epl",
			Market:    "h2h",
			Selection: "Home",
			Stake:     1,
			Odds:      2,
		}); err != nil {
			t.Fatalf("place bet %d: %v", index, err)
		}
	}

	bets, err := betService.Get(context.Background(), request.GetBetsRequest{UserID: "user-1", Limit: 2})
	if err != nil {
		t.Fatalf("get bets: %v", err)
	}
	if len(bets) != 2 {
		t.Fatalf("returned bets = %d, want 2", len(bets))
	}
}

func TestBetServicePlaceUsesAuthoritativeOdds(t *testing.T) {
	db := newTestStore(t)
	walletService := NewWalletService(db)
	if _, err := walletService.Credit(context.Background(), "user-1", 10, "DEPOSIT"); err != nil {
		t.Fatalf("credit wallet: %v", err)
	}

	betService := NewBetService(db, fixedOddsLookup{price: 1.5})
	if _, err := betService.Place(context.Background(), request.PlaceBetRequest{
		UserID: "user-1", EventID: "event-1", SportKey: "soccer_epl", Market: "h2h", Selection: "Home", Stake: 1, Odds: 99,
	}); err != nil {
		t.Fatalf("place bet: %v", err)
	}

	bets, err := betService.Get(context.Background(), request.GetBetsRequest{UserID: "user-1", Limit: 1})
	if err != nil {
		t.Fatalf("get bets: %v", err)
	}
	if len(bets) != 1 || bets[0].Odds != 1.5 {
		t.Fatalf("stored odds = %+v, want provider price 1.5", bets)
	}
}

type fixedOddsLookup struct {
	price float64
}

func (l fixedOddsLookup) GetOdds(_ context.Context, request request.GetOddsRequest) (*response.OddsResponse, error) {
	return &response.OddsResponse{
		ID:       request.EventID,
		SportKey: request.SportKey,
		Bookmakers: []response.Bookmaker{{
			Markets: []response.Market{{
				Key: "h2h", Outcomes: []response.Outcome{{Name: "Home", Price: l.price}},
			}},
		}},
	}, nil
}
