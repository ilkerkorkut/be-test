package client

import (
	"github.com/flotaro/be-test/services/go-svc/config"
	"github.com/flotaro/be-test/services/go-svc/internal/client/oddsapi"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type APIClient interface {
	OddsAPI() oddsapi.Client
}

type apiClient struct {
	oddsAPI oddsapi.Client
}

func NewAPIClient(config config.ConfigurationManager, logger *zap.Logger) APIClient {
	client := resty.New()
	return &apiClient{
		oddsAPI: oddsapi.NewClient(
			client,
			config.GetOddsAPIConfig().APIKey,
			logger,
		),
	}
}

func (c *apiClient) OddsAPI() oddsapi.Client {
	return c.oddsAPI
}
