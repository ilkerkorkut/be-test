package worker

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/database"
	"github.com/flotaro/be-test/services/go-svc/internal/event"
)

func TestPlacedEventSchedulesOneSettlementRequest(t *testing.T) {
	store, err := database.NewJSONStore(filepath.Join(t.TempDir(), "go-svc.json"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	placed := event.BetPlaced{EventID: "placed-1", BetID: "bet-1", UserID: "user-1", PlacedAt: now}
	if err := scheduleSettlement(t.Context(), store, placed); err != nil {
		t.Fatalf("schedule settlement request: %v", err)
	}
	if err := scheduleSettlement(t.Context(), store, placed); err != nil {
		t.Fatalf("schedule duplicate settlement request: %v", err)
	}

	schedules, err := store.GetDueSettlementSchedules(now, 10)
	if err != nil {
		t.Fatalf("get early settlement schedules: %v", err)
	}
	if len(schedules) != 0 {
		t.Fatalf("early settlement schedules = %+v", schedules)
	}
	schedules, err = store.GetDueSettlementSchedules(now.Add(settlementDelay), 10)
	if err != nil {
		t.Fatalf("get due settlement schedules: %v", err)
	}
	if len(schedules) != 1 || schedules[0].ID != "settlement-requested-bet-1" || schedules[0].BetID != placed.BetID || schedules[0].UserID != placed.UserID || !schedules[0].SettleAt.Equal(now.Add(settlementDelay)) {
		t.Fatalf("settlement schedules = %+v", schedules)
	}
}
