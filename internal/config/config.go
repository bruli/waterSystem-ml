package config

import "github.com/caarlos0/env/v11"

type Config struct {
	ServerHost string `env:"SERVER_HOST,required"`
	ModelDir   string `env:"MODEL_DIR,required"`
	PythonPath string `env:"PYTHON_PATH,required"`
}

func New() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
