package oddsapi

import (
	"encoding/json"
	"testing"
)

func TestUpstreamResponsesMapSnakeCaseFieldsToContract(t *testing.T) {
	var sport sportsAPIResponse
	if err := json.Unmarshal([]byte(`{"key":"soccer_epl","group":"Soccer","title":"EPL","active":true,"has_outrights":true}`), &sport); err != nil {
		t.Fatalf("decode sport: %v", err)
	}
	if !sport.HasOutrights {
		t.Fatal("has_outrights was not decoded")
	}

	var odds oddsAPIResponse
	if err := json.Unmarshal([]byte(`{"id":"event-1","sport_key":"soccer_epl","commence_time":"2026-01-01T00:00:00Z","home_team":"Home","away_team":"Away","bookmakers":[]}`), &odds); err != nil {
		t.Fatalf("decode odds: %v", err)
	}
	response := odds.response()
	if response.SportKey != "soccer_epl" || response.CommenceTime == "" || response.HomeTeam != "Home" || response.AwayTeam != "Away" {
		t.Fatalf("unexpected contract response: %+v", response)
	}
}
