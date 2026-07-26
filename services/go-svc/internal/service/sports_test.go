package service

import (
	"context"
	"sync"
	"testing"

	"github.com/flotaro/be-test/services/go-svc/config"
	"github.com/flotaro/be-test/services/go-svc/internal/client/oddsapi"
	"github.com/flotaro/be-test/services/go-svc/internal/response"
)

func TestSportsServiceCachesClientResponseAndReturnsCopies(t *testing.T) {
	upstream := &oddsAPIStub{
		sports: []response.SportsResponse{{Key: "soccer_epl", Title: "Premier League"}},
	}
	service, err := NewSportsService(apiClientStub{oddsAPI: upstream}, config.CacheConfig{})
	if err != nil {
		t.Fatalf("new sports service: %v", err)
	}

	first, err := service.GetSports(context.Background())
	if err != nil {
		t.Fatalf("get first sports response: %v", err)
	}
	first[0].Title = "mutated"

	second, err := service.GetSports(context.Background())
	if err != nil {
		t.Fatalf("get cached sports response: %v", err)
	}
	if upstream.SportsCalls() != 1 {
		t.Fatalf("upstream sports calls = %d, want 1", upstream.SportsCalls())
	}
	if second[0].Title != "Premier League" {
		t.Fatalf("cached response was mutated: %+v", second[0])
	}
}

type apiClientStub struct {
	oddsAPI oddsapi.Client
}

func (c apiClientStub) OddsAPI() oddsapi.Client {
	return c.oddsAPI
}

type oddsAPIStub struct {
	mu          sync.Mutex
	sports      []response.SportsResponse
	sportsCalls int
}

func (c *oddsAPIStub) GetSports(_ context.Context) ([]response.SportsResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sportsCalls++
	return append([]response.SportsResponse(nil), c.sports...), nil
}

func (c *oddsAPIStub) GetEvents(_ context.Context, _ string) ([]response.EventResponse, error) {
	return nil, nil
}

func (c *oddsAPIStub) GetOdds(_ context.Context, _ string, _ string) (*response.OddsResponse, error) {
	return nil, nil
}

func (c *oddsAPIStub) SportsCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.sportsCalls
}
