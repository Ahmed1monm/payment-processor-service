package config

import (
	"os"
	"strconv"
)

// Config holds application level configuration loaded from environment variables.
type Config struct {
	ServerPort  string
	MySQLDSN    string
	RedisAddr   string
	RedisDB     int
	RedisPass   string
	JWTSecret   string
	SwaggerHost string
}

// Load builds Config from environment with sensible defaults.
func Load() *Config {
	return &Config{
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		MySQLDSN:    getEnv("MYSQL_DSN", "user:password@tcp(localhost:3306)/app?charset=utf8mb4&parseTime=True&loc=Local"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		RedisDB:     getEnvInt("REDIS_DB", 0),
		RedisPass:   os.Getenv("REDIS_PASSWORD"),
		JWTSecret:   getEnv("JWT_SECRET", "change-me"),
		SwaggerHost: os.Getenv("SWAGGER_HOST"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return def
}
