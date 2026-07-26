package config

type Configuration struct {
	App      AppConfig
	Auth     AuthConfig
	Odds     OddsAPIConfig
	Cache    CacheConfig
	Database DatabaseConfig
	Kafka    KafkaConfig
	Data     DataConfig
}

type AppConfig struct {
	Name     string `env:"APP_NAME" envDefault:"go-svc"`
	Port     string `env:"PORT" envDefault:"8081"`
	Env      string `env:"APP_ENVIRONMENT" envDefault:"development"`
	LogLevel string `env:"APP_LOG_LEVEL" envDefault:"info"`
}

type AuthConfig struct {
	JWTSecret string `env:"JWT_SECRET,required"`
}

type OddsAPIConfig struct {
	APIKey string `env:"ODDS_API_KEY,required"`
}

type CacheConfig struct {
	RedisURL string `env:"REDIS_URL"`
}

type DatabaseConfig struct {
	Path string `env:"DATABASE_PATH" envDefault:"../../data/go-svc.json"`
}

type KafkaConfig struct {
	Brokers       string `env:"KAFKA_BROKERS" envDefault:"localhost:39092"`
	ConsumerGroup string `env:"KAFKA_CONSUMER_GROUP" envDefault:"go-svc-settlement-worker"`
}

type DataConfig struct {
	FolderPath string `env:"DATA_FOLDER_PATH" envDefault:"../../data"`
}
