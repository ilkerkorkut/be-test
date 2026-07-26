package enums

type BetStatus string

const (
	BetStatusPending  BetStatus = "PENDING"
	BetStatusAccepted BetStatus = "ACCEPTED"
	BetStatusRejected BetStatus = "REJECTED"
	BetStatusWon      BetStatus = "WON"
	BetStatusLost     BetStatus = "LOST"
)
