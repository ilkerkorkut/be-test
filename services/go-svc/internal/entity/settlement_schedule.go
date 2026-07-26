package entity

import "time"

type SettlementSchedule struct {
	ID           string     `json:"id"`
	BetID        string     `json:"betId"`
	UserID       string     `json:"userId"`
	SettleAt     time.Time  `json:"settleAt"`
	DispatchedAt *time.Time `json:"dispatchedAt,omitempty"`
}
