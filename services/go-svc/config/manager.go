package config

import (
	"log"

	"github.com/caarlos0/env/v6"
)

type ConfigurationManager interface {
	GetAppConfig() AppConfig
	GetAuthConfig() AuthConfig
	GetOddsAPIConfig() OddsAPIConfig
	GetCacheConfig() CacheConfig
	GetDatabaseConfig() DatabaseConfig
	GetKafkaConfig() KafkaConfig
	GetDataConfig() DataConfig
}

type configurationManager struct {
	App      AppConfig
	Auth     AuthConfig
	Odds     OddsAPIConfig
	Cache    CacheConfig
	Database DatabaseConfig
	Kafka    KafkaConfig
	Data     DataConfig
}

func NewConfigurationManager() ConfigurationManager {
	var configuration Configuration

	if err := env.Parse(&configuration); err != nil {
		log.Fatalf("Error parsing configuration: %v", err)
	}

	return &configurationManager{
		App:      configuration.App,
		Auth:     configuration.Auth,
		Odds:     configuration.Odds,
		Cache:    configuration.Cache,
		Database: configuration.Database,
		Kafka:    configuration.Kafka,
		Data:     configuration.Data,
	}
}

func (c *configurationManager) GetOddsAPIConfig() OddsAPIConfig {
	return c.Odds
}

func (c *configurationManager) GetCacheConfig() CacheConfig {
	return c.Cache
}

func (c *configurationManager) GetDatabaseConfig() DatabaseConfig {
	return c.Database
}

func (c *configurationManager) GetKafkaConfig() KafkaConfig {
	return c.Kafka
}

func (c *configurationManager) GetAppConfig() AppConfig {
	return c.App
}

func (c *configurationManager) GetAuthConfig() AuthConfig {
	return c.Auth
}

func (c *configurationManager) GetDataConfig() DataConfig {
	return c.Data
}
