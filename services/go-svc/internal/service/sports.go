package service

import (
	"context"
	"fmt"
	"time"

	"github.com/flotaro/be-test/services/go-svc/config"
	"github.com/flotaro/be-test/services/go-svc/internal/client"
	"github.com/flotaro/be-test/services/go-svc/internal/request"
	"github.com/flotaro/be-test/services/go-svc/internal/response"
	"github.com/flotaro/be-test/services/go-svc/pkg/cache"
)

const (
	sportsCacheTTL = 5 * time.Minute
	eventsCacheTTL = 30 * time.Second
	oddsCacheTTL   = 15 * time.Second
)

type SportsService interface {
	GetSports(ctx context.Context) ([]response.SportsResponse, error)
	GetEvents(ctx context.Context, req request.GetEventsRequest) ([]response.EventResponse, error)
	GetOdds(ctx context.Context, req request.GetOddsRequest) (*response.OddsResponse, error)
}

type sportsService struct {
	client      client.APIClient
	sportsCache cache.Store[[]response.SportsResponse]
	eventsCache cache.Store[[]response.EventResponse]
	oddsCache   cache.Store[*response.OddsResponse]
}

func NewSportsService(client client.APIClient, config config.CacheConfig) (SportsService, error) {
	service := &sportsService{
		client:      client,
		sportsCache: cache.NewMemory[[]response.SportsResponse](),
		eventsCache: cache.NewMemory[[]response.EventResponse](),
		oddsCache:   cache.NewMemory[*response.OddsResponse](),
	}
	if config.RedisURL == "" {
		return service, nil
	}

	redisClient, err := cache.NewRedisClient(config.RedisURL)
	if err != nil {
		return nil, err
	}

	service.sportsCache = cache.NewRedis[[]response.SportsResponse](redisClient, "go-svc:sports")
	service.eventsCache = cache.NewRedis[[]response.EventResponse](redisClient, "go-svc:events")
	service.oddsCache = cache.NewRedis[*response.OddsResponse](redisClient, "go-svc:odds")

	return service, nil
}

func (s *sportsService) GetSports(ctx context.Context) ([]response.SportsResponse, error) {
	sports, err := s.sportsCache.GetOrLoad(ctx, "sports", sportsCacheTTL, func() ([]response.SportsResponse, error) {
		return s.client.OddsAPI().GetSports(ctx)
	})
	if err != nil {
		return nil, err
	}

	return cloneSports(sports), nil
}

func (s *sportsService) GetEvents(ctx context.Context, req request.GetEventsRequest) ([]response.EventResponse, error) {
	if req.SportKey == "" {
		return nil, fmt.Errorf("sport key is required")
	}

	events, err := s.eventsCache.GetOrLoad(ctx, "events:"+req.SportKey, eventsCacheTTL, func() ([]response.EventResponse, error) {
		return s.client.OddsAPI().GetEvents(ctx, req.SportKey)
	})
	if err != nil {
		return nil, err
	}

	return cloneEvents(events), nil
}

func (s *sportsService) GetOdds(ctx context.Context, req request.GetOddsRequest) (*response.OddsResponse, error) {
	if req.SportKey == "" || req.EventID == "" {
		return nil, fmt.Errorf("sport key and event id are required")
	}

	odds, err := s.oddsCache.GetOrLoad(ctx, "odds:"+req.SportKey+":"+req.EventID, oddsCacheTTL, func() (*response.OddsResponse, error) {
		return s.client.OddsAPI().GetOdds(ctx, req.SportKey, req.EventID)
	})
	if err != nil {
		return nil, err
	}

	return cloneOdds(odds), nil
}

func cloneSports(sports []response.SportsResponse) []response.SportsResponse {
	return append([]response.SportsResponse(nil), sports...)
}

func cloneEvents(events []response.EventResponse) []response.EventResponse {
	return append([]response.EventResponse(nil), events...)
}

func cloneOdds(odds *response.OddsResponse) *response.OddsResponse {
	if odds == nil {
		return nil
	}

	copyOdds := *odds
	copyOdds.Bookmakers = make([]response.Bookmaker, len(odds.Bookmakers))
	for bookmakerIndex, bookmaker := range odds.Bookmakers {
		copyBookmaker := bookmaker
		copyBookmaker.Markets = make([]response.Market, len(bookmaker.Markets))
		for marketIndex, market := range bookmaker.Markets {
			copyMarket := market
			copyMarket.Outcomes = append([]response.Outcome(nil), market.Outcomes...)
			copyBookmaker.Markets[marketIndex] = copyMarket
		}
		copyOdds.Bookmakers[bookmakerIndex] = copyBookmaker
	}

	return &copyOdds
}
