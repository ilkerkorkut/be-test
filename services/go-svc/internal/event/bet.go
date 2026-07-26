package event

import "time"

const (
	TopicBetsPlaced              = "bets.placed.v1"
	TopicBetsSettlementRequested = "bets.settlement.requested.v1"
	TopicBetsSettled             = "bets.settled.v1"
)

type BetPlaced struct {
	EventID  string    `json:"eventId"`
	BetID    string    `json:"betId"`
	UserID   string    `json:"userId"`
	PlacedAt time.Time `json:"placedAt"`
}

type BetSettlementRequested struct {
	EventID     string    `json:"eventId"`
	BetID       string    `json:"betId"`
	UserID      string    `json:"userId"`
	RequestedAt time.Time `json:"requestedAt"`
}

type BetSettled struct {
	EventID   string    `json:"eventId"`
	BetID     string    `json:"betId"`
	UserID    string    `json:"userId"`
	Status    string    `json:"status"`
	Payout    float64   `json:"payout"`
	SettledAt time.Time `json:"settledAt"`
}
