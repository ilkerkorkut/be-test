package oddsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/flotaro/be-test/services/go-svc/internal/response"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

const baseURL = "https://api.the-odds-api.com/v4"

type Client interface {
	GetSports(ctx context.Context) ([]response.SportsResponse, error)
	GetEvents(ctx context.Context, sportKey string) ([]response.EventResponse, error)
	GetOdds(ctx context.Context, sportKey string, eventID string) (*response.OddsResponse, error)
}

type client struct {
	restyClient *resty.Client
	apiKey      string
	logger      *zap.Logger
}

func NewClient(restyClient *resty.Client, apiKey string, logger *zap.Logger) Client {
	restyClient.SetBaseURL(baseURL)

	return &client{
		restyClient: restyClient,
		apiKey:      apiKey,
		logger:      logger,
	}
}

func (c *client) GetSports(ctx context.Context) ([]response.SportsResponse, error) {
	value, err := c.get(ctx, "/sports", nil)
	if err != nil {
		return nil, err
	}

	var sports []sportsAPIResponse
	if err = json.Unmarshal(value, &sports); err != nil {
		return nil, fmt.Errorf("decode sports response: %w", err)
	}

	responses := make([]response.SportsResponse, 0, len(sports))
	for _, sport := range sports {
		responses = append(responses, response.SportsResponse{
			Key:          sport.Key,
			Group:        sport.Group,
			Title:        sport.Title,
			Active:       sport.Active,
			HasOutrights: sport.HasOutrights,
		})
	}

	return responses, nil
}

func (c *client) GetEvents(ctx context.Context, sportKey string) ([]response.EventResponse, error) {
	value, err := c.get(ctx, "/sports/"+sportKey+"/events", nil)
	if err != nil {
		return nil, err
	}

	var events []eventAPIResponse
	if err = json.Unmarshal(value, &events); err != nil {
		return nil, fmt.Errorf("decode events response: %w", err)
	}

	responses := make([]response.EventResponse, 0, len(events))
	for _, event := range events {
		responses = append(responses, event.response())
	}

	return responses, nil
}

func (c *client) GetOdds(ctx context.Context, sportKey string, eventID string) (*response.OddsResponse, error) {
	value, err := c.get(
		ctx,
		"/sports/"+sportKey+"/events/"+eventID+"/odds",
		map[string]string{"markets": "h2h", "regions": "eu"},
	)
	if err != nil {
		return nil, err
	}

	var odds oddsAPIResponse
	if err = json.Unmarshal(value, &odds); err != nil {
		return nil, fmt.Errorf("decode odds response: %w", err)
	}

	return odds.response(), nil
}

func (c *client) get(ctx context.Context, path string, query map[string]string) ([]byte, error) {
	request := c.restyClient.R().SetContext(ctx).SetQueryParam("apiKey", c.apiKey)
	for key, value := range query {
		request.SetQueryParam(key, value)
	}

	apiResponse, err := request.Get(path)
	if err != nil {
		return nil, fmt.Errorf("request odds api: %w", err)
	}

	c.logger.Info("odds api quota",
		zap.String("path", path),
		zap.String("requests_remaining", apiResponse.Header().Get("x-requests-remaining")),
		zap.String("requests_used", apiResponse.Header().Get("x-requests-used")),
	)

	if apiResponse.StatusCode() < http.StatusOK || apiResponse.StatusCode() >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("odds api returned status %d", apiResponse.StatusCode())
	}

	return append([]byte(nil), apiResponse.Body()...), nil
}

type sportsAPIResponse struct {
	Key          string `json:"key"`
	Group        string `json:"group"`
	Title        string `json:"title"`
	Active       bool   `json:"active"`
	HasOutrights bool   `json:"has_outrights"`
}

type eventAPIResponse struct {
	ID           string `json:"id"`
	SportKey     string `json:"sport_key"`
	CommenceTime string `json:"commence_time"`
	HomeTeam     string `json:"home_team"`
	AwayTeam     string `json:"away_team"`
}

func (e eventAPIResponse) response() response.EventResponse {
	return response.EventResponse{
		ID:           e.ID,
		SportKey:     e.SportKey,
		CommenceTime: e.CommenceTime,
		HomeTeam:     e.HomeTeam,
		AwayTeam:     e.AwayTeam,
	}
}

type oddsAPIResponse struct {
	eventAPIResponse
	Bookmakers []response.Bookmaker `json:"bookmakers"`
}

func (o oddsAPIResponse) response() *response.OddsResponse {
	return &response.OddsResponse{
		ID:           o.ID,
		SportKey:     o.SportKey,
		CommenceTime: o.CommenceTime,
		HomeTeam:     o.HomeTeam,
		AwayTeam:     o.AwayTeam,
		Bookmakers:   o.Bookmakers,
	}
}
