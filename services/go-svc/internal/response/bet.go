package response

import (
	"github.com/flotaro/be-test/services/go-svc/internal/enums"
)

type BetResponse struct {
	BetID  string          `json:"betId"`
	Status enums.BetStatus `json:"status"`
}

type BetHistoryResponse struct {
	ID        string          `json:"id"`
	EventID   string          `json:"eventId"`
	Market    string          `json:"market"`
	Selection string          `json:"selection"`
	Stake     float64         `json:"stake"`
	Odds      float64         `json:"odds"`
	Status    enums.BetStatus `json:"status"`
	Payout    *float64        `json:"payout"`
	PlacedAt  string          `json:"placedAt"`
	SettledAt *string         `json:"settledAt"`
}
