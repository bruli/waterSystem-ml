package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ServerHost       string  `env:"SERVER_HOST,required"`
	ModelDir         string  `env:"MODEL_DIR,required"`
	PythonPath       string  `env:"PYTHON_PATH,required"`
	Env              string  `env:"ENV" envDefault:"PROD"`
	InfluxDBURL      string  `env:"INFLUXDB_URL,required"`
	InfluxDBToken    string  `env:"INFLUXDB_TOKEN,required"`
	InfluxDBOrg      string  `env:"INFLUXDB_ORG,required"`
	InfluxDBBucket   string  `env:"INFLUXDB_BUCKET,required"`
	WaterSystemHost  string  `env:"WATER_SYSTEM_HOST,required"`
	WaterSystemPort  string  `env:"WATER_SYSTEM_PORT,required"`
	WaterSystemToken string  `env:"WATER_SYSTEM_TOKEN,required"`
	LogLevel         string  `env:"LOG_LEVEL,required"`
	NtfyURL          string  `env:"NTFY_URL,required"`
	NtfyTopic        string  `env:"NTFY_TOPIC,required"`
	NtfyUser         string  `env:"NTFY_USER,required"`
	NtfyPassword     string  `env:"NTFY_PASSWORD,required"`
	BonsaiBigV100    float64 `env:"BONSAI_BIG_V100,required"`
	BonsaiBigV40     float64 `env:"BONSAI_BIG_V40,required"`
	BonsaiSmallV100  float64 `env:"BONSAI_SMALL_V100,required"`
	BonsaiSmallV40   float64 `env:"BONSAI_SMALL_V40,required"`
	PostgresDatabase string  `env:"POSTGRES_DATABASE,required"`
	PostgresHost     string  `env:"POSTGRES_HOST,required"`
	PostgresPort     string  `env:"POSTGRES_PORT,required"`
	PostgresUser     string  `env:"POSTGRES_USER,required"`
	PostgresPassword string  `env:"POSTGRES_PASSWORD,required"`
}

func New() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) IsProd() bool {
	return c.Env == "PROD"
}

func (c *Config) PostgresDataSource() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", c.PostgresUser, c.PostgresPassword, c.PostgresHost, c.PostgresPort, c.PostgresDatabase)
}
