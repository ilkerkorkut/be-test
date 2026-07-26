package worker

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/entity"
	"github.com/flotaro/be-test/services/go-svc/internal/event"
	"go.uber.org/zap"
)

func TestSettlementSchedulerDispatchesOnlyDueRequests(t *testing.T) {
	store, err := database.NewJSONStore(filepath.Join(t.TempDir(), "go-svc.json"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	schedule := entity.SettlementSchedule{ID: "settlement-requested-bet-1", BetID: "bet-1", UserID: "user-1", SettleAt: now.Add(settlementDelay)}
	if err := store.ScheduleSettlement(schedule); err != nil {
		t.Fatalf("schedule settlement: %v", err)
	}

	scheduler := NewSettlementScheduler(store, zap.NewNop())
	if err := scheduler.Dispatch(now); err != nil {
		t.Fatalf("dispatch early schedule: %v", err)
	}
	events, err := store.GetUnpublishedOutbox(10)
	if err != nil {
		t.Fatalf("get early outbox: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("early outbox events = %+v", events)
	}

	if err := scheduler.Dispatch(schedule.SettleAt); err != nil {
		t.Fatalf("dispatch due schedule: %v", err)
	}
	events, err = store.GetUnpublishedOutbox(10)
	if err != nil {
		t.Fatalf("get due outbox: %v", err)
	}
	if len(events) != 1 || events[0].Topic != event.TopicBetsSettlementRequested {
		t.Fatalf("due outbox events = %+v", events)
	}
	var requested event.BetSettlementRequested
	if err := json.Unmarshal(events[0].Payload, &requested); err != nil {
		t.Fatalf("decode settlement request: %v", err)
	}
	if requested.EventID != schedule.ID || requested.BetID != schedule.BetID || requested.UserID != schedule.UserID {
		t.Fatalf("settlement request = %+v", requested)
	}

	if err := scheduler.Dispatch(schedule.SettleAt); err != nil {
		t.Fatalf("dispatch duplicate due schedule: %v", err)
	}
	events, err = store.GetUnpublishedOutbox(10)
	if err != nil {
		t.Fatalf("get duplicate outbox: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("outbox events after duplicate dispatch = %+v", events)
	}
}

func TestSettlementSchedulerReconcilesPendingBets(t *testing.T) {
	store, err := database.NewJSONStore(filepath.Join(t.TempDir(), "go-svc.json"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	payload, err := json.Marshal(event.BetPlaced{EventID: "placed-1", BetID: "bet-1", UserID: "user-1", PlacedAt: now})
	if err != nil {
		t.Fatalf("encode placed event: %v", err)
	}
	if _, err := store.CreditWallet("user-1", 1000, entity.Ledger{ID: "deposit", UserID: "user-1", Type: "DEPOSIT", Amount: 1000, CreatedAt: now}); err != nil {
		t.Fatalf("credit wallet: %v", err)
	}
	if err := store.CreateBet(
		entity.Bet{ID: "bet-1", UserID: "user-1", StakeCents: 500, Odds: 2, Status: "PENDING", PlacedAt: now},
		entity.Ledger{ID: "debit", UserID: "user-1", Type: "BET_DEBIT", Amount: -500, CreatedAt: now},
		entity.OutboxEvent{ID: "placed-1", Topic: event.TopicBetsPlaced, Key: "bet-1", Payload: payload, CreatedAt: now},
	); err != nil {
		t.Fatalf("create bet: %v", err)
	}

	scheduler := NewSettlementScheduler(store, zap.NewNop())
	if err := scheduler.Reconcile(); err != nil {
		t.Fatalf("reconcile pending bets: %v", err)
	}
	schedules, err := store.GetDueSettlementSchedules(now.Add(settlementDelay), 10)
	if err != nil {
		t.Fatalf("get due settlement schedules: %v", err)
	}
	if len(schedules) != 1 || schedules[0].ID != settlementScheduleID("bet-1") {
		t.Fatalf("settlement schedules = %+v", schedules)
	}
}
