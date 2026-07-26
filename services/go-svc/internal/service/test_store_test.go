package service

import (
	"path/filepath"
	"testing"

	"github.com/flotaro/be-test/services/go-svc/internal/database"
)

func newTestStore(t *testing.T) database.Store {
	t.Helper()

	store, err := database.NewJSONStore(filepath.Join(t.TempDir(), "go-svc.json"))
	if err != nil {
		t.Fatalf("create test store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	return store
}
