package entity

import "time"

type Wallet struct {
	UserID       string `json:"userId"`
	BalanceCents int64  `json:"balanceCents"`
	Currency     string `json:"currency"`
}

type Ledger struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Type      string    `json:"type"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"createdAt"`
}
