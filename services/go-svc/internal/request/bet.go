package request

type PlaceBetRequest struct {
	UserID    string  `json:"userId"`
	EventID   string  `json:"eventId"`
	SportKey  string  `json:"sportKey"`
	Market    string  `json:"market"`
	Selection string  `json:"selection"`
	Stake     float64 `json:"stake"`
	Odds      float64 `json:"odds"`
}

type GetBetsRequest struct {
	UserID string `query:"userId"`
	Limit  int    `query:"limit"`
}
