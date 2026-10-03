package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config aggregates all operational configurations for the application.
type Config struct {
	Server ServerConfig
	DB     DBConfig
	Redis  RedisConfig
	Worker WorkerConfig
}

type ServerConfig struct {
	Env          string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	BaseURL      string
}

type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DSN returns the formatted PostgreSQL connection string.
func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}

type RedisConfig struct {
	Host       string
	Port       string
	Password   string
	DB         int
	DefaultTTL time.Duration
}

// Addr returns host:port address string for Redis client.
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

type WorkerConfig struct {
	PoolSize      int
	BufferSize    int
	BatchSize     int
	FlushInterval time.Duration
}

// Load reads settings from the local .env file (if present) and overrides with OS environment variables.
func Load() (*Config, error) {
	// Best-effort load .env; in container/production, environment is passed by container runtime.
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Env:          getEnv("SERVER_ENV", "development"),
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
			BaseURL:      getEnv("BASE_URL", "http://localhost:8080"),
		},
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Name:            getEnv("DB_NAME", "shortgo"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:       getEnv("REDIS_HOST", "localhost"),
			Port:       getEnv("REDIS_PORT", "6379"),
			Password:   getEnv("REDIS_PASSWORD", ""),
			DB:         getIntEnv("REDIS_DB", 0),
			DefaultTTL: getDurationEnv("REDIS_DEFAULT_TTL", 24*time.Hour),
		},
		Worker: WorkerConfig{
			PoolSize:      getIntEnv("WORKER_POOL_SIZE", 5),
			BufferSize:    getIntEnv("WORKER_BUFFER_SIZE", 10000),
			BatchSize:     getIntEnv("WORKER_BATCH_SIZE", 100),
			FlushInterval: getDurationEnv("WORKER_FLUSH_INTERVAL", 2*time.Second),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getIntEnv(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}
