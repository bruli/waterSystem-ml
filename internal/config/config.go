package config

import "github.com/caarlos0/env/v11"

type Config struct {
	ServerHost       string `env:"SERVER_HOST,required"`
	ModelDir         string `env:"MODEL_DIR,required"`
	PythonPath       string `env:"PYTHON_PATH,required"`
	TelegramToken    string `env:"TELEGRAM_TOKEN,required"`
	TelegramChatID   int    `env:"TELEGRAM_CHAT_ID,required"`
	Env              string `env:"ENV" envDefault:"PROD"`
	InfluxDBURL      string `env:"INFLUXDB_URL,required"`
	InfluxDBToken    string `env:"INFLUXDB_TOKEN,required"`
	InfluxDBOrg      string `env:"INFLUXDB_ORG,required"`
	InfluxDBBucket   string `env:"INFLUXDB_BUCKET,required"`
	WaterSystemHost  string `env:"WATER_SYSTEM_HOST,required"`
	WaterSystemPort  string `env:"WATER_SYSTEM_PORT,required"`
	WaterSystemToken string `env:"WATER_SYSTEM_TOKEN,required"`
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
