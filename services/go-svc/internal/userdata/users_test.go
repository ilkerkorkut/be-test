package userdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadInitialWallets(t *testing.T) {
	folderPath := t.TempDir()
	content := `[{"id":"user-1","currency":"CAD","initialBalance":4230.2,"createdAt":"2026-07-25T00:00:00Z"}]`
	if err := os.WriteFile(filepath.Join(folderPath, "users.json"), []byte(content), 0o600); err != nil {
		t.Fatalf("write users data: %v", err)
	}

	wallets, err := LoadInitialWallets(folderPath)
	if err != nil {
		t.Fatalf("load initial wallets: %v", err)
	}
	wallet := wallets["user-1"]
	if wallet.Currency != "CAD" || wallet.BalanceCents != 423020 {
		t.Fatalf("wallet = %+v", wallet)
	}
}
