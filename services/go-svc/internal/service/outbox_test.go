package service

import (
	"context"
	"testing"

	"github.com/flotaro/be-test/services/go-svc/internal/request"
	"go.uber.org/zap"
)

func TestOutboxDispatcherPublishesAndMarksEvent(t *testing.T) {
	db := newTestStore(t)
	walletService := NewWalletService(db)
	if _, err := walletService.Credit(context.Background(), "user-1", 20, "DEPOSIT"); err != nil {
		t.Fatalf("credit wallet: %v", err)
	}
	if _, err := NewBetService(db).Place(context.Background(), request.PlaceBetRequest{UserID: "user-1", EventID: "event-1", SportKey: "soccer_epl", Market: "h2h", Selection: "Home", Stake: 5, Odds: 2}); err != nil {
		t.Fatalf("place bet: %v", err)
	}

	publisher := &publisherStub{}
	if err := NewOutboxDispatcher(db, publisher, zap.NewNop()).Dispatch(context.Background()); err != nil {
		t.Fatalf("dispatch outbox: %v", err)
	}
	if publisher.publishes != 1 {
		t.Fatalf("publishes = %d, want 1", publisher.publishes)
	}
	events, err := db.GetUnpublishedOutbox(10)
	if err != nil {
		t.Fatalf("get unpublished events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("unpublished events = %d, want 0", len(events))
	}
}

type publisherStub struct {
	publishes int
}

func (p *publisherStub) Publish(_ context.Context, _ string, _ string, _ []byte) error {
	p.publishes++
	return nil
}
