package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database DatabaseConfig
	Bot      BotConfig
	API      APIConfig
}

type DatabaseConfig struct {
	Host                  string
	Port                  int
	User                  string
	Password              string
	Database              string
	SSLMode               string
	MaxOpenConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Database, d.SSLMode)
}

type BotConfig struct {
	Token    string
	Debug    bool
	AdminIds []string
	Timeout  int
}

type APIConfig struct {
	BaseURL string
	APIKey  string
	Enabled bool
}

func Load() (*Config, error) {
	_ = godotenv.Load(".env")

	adminEnv := os.Getenv("TELEGRAM_ADMIN_IDS")
	var admins []string
	if adminEnv != "" {
		admins = strings.Split(adminEnv, ",")
	}

	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	maxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNECTIONS", "25"))
	maxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNECTIONS", "25"))
	timeout, _ := strconv.Atoi(getEnv("TELEGRAM_TIMEOUT", "60"))
	apiEnabled := getEnv("API_ENABLED", "false") == "true"

	return &Config{
		Database: DatabaseConfig{
			Host:                  getEnv("DB_HOST", "localhost"),
			Port:                  dbPort,
			User:                  getEnv("DB_USER", "postgres"),
			Password:              getEnv("DB_PASSWORD", "password"),
			Database:              getEnv("DB_NAME", "repair_bot"),
			SSLMode:               getEnv("DB_SSLMODE", "disable"),
			MaxOpenConnections:    maxOpenConns,
			MaxIdleConnections:    maxIdleConns,
			ConnectionMaxLifetime: 0,
		},
		Bot: BotConfig{
			Token:    getEnv("TELEGRAM_BOT_TOKEN", ""),
			Debug:    getEnv("TELEGRAM_DEBUG", "false") == "true",
			AdminIds: admins,
			Timeout:  timeout,
		},
		API: APIConfig{
			BaseURL: getEnv("API_BASE_URL", "http://localhost:8080"),
			APIKey:  getEnv("API_KEY", ""),
			Enabled: apiEnabled,
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
