package entity

import (
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/enums"
)

type Bet struct {
	ID          string          `json:"id"`
	UserID      string          `json:"userId"`
	EventID     string          `json:"eventId"`
	SportKey    string          `json:"sportKey"`
	Market      string          `json:"market"`
	Selection   string          `json:"selection"`
	StakeCents  int64           `json:"stakeCents"`
	Odds        float64         `json:"odds"`
	Status      enums.BetStatus `json:"status"`
	PayoutCents *int64          `json:"payoutCents,omitempty"`
	PlacedAt    time.Time       `json:"placedAt"`
	SettledAt   *time.Time      `json:"settledAt,omitempty"`
}
