package userdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/database"
)

type user struct {
	ID             string  `json:"id"`
	Currency       string  `json:"currency"`
	InitialBalance float64 `json:"initialBalance"`
	CreatedAt      string  `json:"createdAt"`
}

func LoadInitialWallets(folderPath string) (map[string]database.InitialWallet, error) {
	content, err := os.ReadFile(filepath.Join(folderPath, "users.json"))
	if err != nil {
		return nil, fmt.Errorf("read users data: %w", err)
	}

	var users []user
	if err := json.Unmarshal(content, &users); err != nil {
		return nil, fmt.Errorf("decode users data: %w", err)
	}

	wallets := make(map[string]database.InitialWallet, len(users))
	for _, value := range users {
		if value.ID == "" || strings.TrimSpace(value.Currency) == "" {
			return nil, errors.New("users data contains a user without id or currency")
		}
		if value.InitialBalance < 0 || math.IsNaN(value.InitialBalance) || math.IsInf(value.InitialBalance, 0) {
			return nil, fmt.Errorf("user %s has an invalid initial balance", value.ID)
		}
		createdAt, err := time.Parse(time.RFC3339Nano, value.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse created time for user %s: %w", value.ID, err)
		}
		wallets[value.ID] = database.InitialWallet{
			UserID:       value.ID,
			Currency:     value.Currency,
			BalanceCents: int64(math.Round(value.InitialBalance * 100)),
			CreatedAt:    createdAt.UTC(),
		}
	}
	return wallets, nil
}
