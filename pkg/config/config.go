package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
}

type ServerConfig struct {
	Host string
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	AccessSecret     string
	RefreshSecret    string
	AccessExpiryMin  int
	RefreshExpiryMin int
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "chattergo"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			AccessSecret:     getEnv("JWT_ACCESS_SECRET", "your-access-secret-key"),
			RefreshSecret:    getEnv("JWT_REFRESH_SECRET", "your-refresh-secret-key"),
			AccessExpiryMin:  getEnvAsInt("JWT_ACCESS_EXPIRY_MIN", 15),
			RefreshExpiryMin: getEnvAsInt("JWT_REFRESH_EXPIRY_MIN", 10080), // 7 days
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func (c *JWTConfig) GetAccessExpiry() time.Duration {
	return time.Duration(c.AccessExpiryMin) * time.Minute
}

func (c *JWTConfig) GetRefreshExpiry() time.Duration {
	return time.Duration(c.RefreshExpiryMin) * time.Minute
}
