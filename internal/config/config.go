package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Argon2   Argon2Config
	CORS     CORSConfig
	Monitoring MonitoringConfig
}

type ServerConfig struct {
	Port            string
	Env             string
	SeedAdminSecret string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	DSN      string
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type Argon2Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

type CORSConfig struct {
	Origins []string
}

type MonitoringConfig struct {
	PrometheusEnabled bool
}

func Load() (*Config, error) {
	godotenv.Load()

	accessExpiry, _ := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	refreshExpiry, _ := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPass := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "school_system")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName)

	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("PORT", "8080"),
			Env:             getEnv("ENV", "development"),
			SeedAdminSecret: getEnv("SEED_ADMIN_SECRET", ""),
		},
		Database: DatabaseConfig{
			Host:     dbHost,
			Port:     dbPort,
			User:     dbUser,
			Password: dbPass,
			Name:     dbName,
			DSN:      dsn,
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", ""),
			AccessExpiry:  accessExpiry,
			RefreshExpiry: refreshExpiry,
		},
		Argon2: Argon2Config{
			Memory:      uint32(getEnvInt("ARGON2_MEMORY", 65536)),
			Iterations:  uint32(getEnvInt("ARGON2_ITERATIONS", 3)),
			Parallelism: uint8(getEnvInt("ARGON2_PARALLELISM", 2)),
			SaltLength:  uint32(getEnvInt("ARGON2_SALT_LENGTH", 16)),
			KeyLength:   uint32(getEnvInt("ARGON2_KEY_LENGTH", 32)),
		},
		CORS: CORSConfig{
			Origins: []string{getEnv("CORS_ORIGINS", "http://localhost:5173")},
		},
		Monitoring: MonitoringConfig{
			PrometheusEnabled: getEnv("PROMETHEUS_ENABLED", "true") == "true",
		},
	}

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
