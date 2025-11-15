package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HTTPPort string `envconfig:"HTTP_PORT" default:"8080"`

	DatabaseHost     string `envconfig:"DB_HOST" required:"true"`
	DatabasePort     string `envconfig:"DB_PORT" default:"5432"`
	DatabaseUser     string `envconfig:"DB_USER" required:"true"`
	DatabasePassword string `envconfig:"DB_PASSWORD" required:"true"`
	DatabaseName     string `envconfig:"DB_NAME" required:"true"`
	DatabaseSSLMode  string `envconfig:"DB_SSLMODE" default:"disable"`

	AdminToken string `envconfig:"ADMIN_TOKEN" required:"true"`

	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}

// GetDSN возвращает строку подключения к базе данных
func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DatabaseUser,
		c.DatabasePassword,
		c.DatabaseHost,
		c.DatabasePort,
		c.DatabaseName,
		c.DatabaseSSLMode,
	)
}
