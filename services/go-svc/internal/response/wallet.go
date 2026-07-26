package response

type WalletResponse struct {
	Balance  float64          `json:"balance"`
	Currency string           `json:"currency"`
	Ledger   []LedgerResponse `json:"ledger"`
}

type LedgerResponse struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	CreatedAt string  `json:"createdAt"`
}
