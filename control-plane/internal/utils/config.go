// control-plane/internal/utils/config.go
package utils

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
	RabbitMQ    RabbitMQConfig
	JWT         JWTConfig
	LogLevel    string
	App         AppConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	Database     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

type RabbitMQConfig struct {
	URL      string
	Host     string
	Port     int
	User     string
	Password string
	VHost    string
}

type AppConfig struct {
	Name string
	Salt string
}

type JWTConfig struct {
	Secret                 string
	AccessTokenExpiration  int // ms
	RefreshTokenExpiration int // ms
}

func LoadConfig() (*Config, error) {
	// Load .env file
	_ = godotenv.Load()

	config := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		App: AppConfig{
			Name: getEnv("APP_NAME", "gon-cloud-platform"),
			Salt: getEnv("APP_SALT", "gcp-salt"),
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnvAsInt("DB_PORT", 5432),
			User:         getEnv("DB_USER", "gcp_user"),
			Password:     getEnv("DB_PASSWORD", "gcp_password"),
			Database:     getEnv("DB_NAME", "gcp_db"),
			SSLMode:      getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns: getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		},
		RabbitMQ: RabbitMQConfig{
			Host:     getEnv("RABBITMQ_HOST", "localhost"),
			Port:     getEnvAsInt("RABBITMQ_PORT", 5672),
			User:     getEnv("RABBITMQ_USER", "guest"),
			Password: getEnv("RABBITMQ_PASSWORD", "guest"),
			VHost:    getEnv("RABBITMQ_VHOST", "/"),
		},
		JWT: JWTConfig{
			Secret:                 getEnv("JWT_SECRET", "your-secret-key"),
			AccessTokenExpiration:  getEnvAsInt("ACCESS_TOKEN_EXPIRATION", 1800000),    // 30 Minuutes
			RefreshTokenExpiration: getEnvAsInt("REFRESH_TOKEN_EXPIRATION", 604800000), // 7 Days
		},
	}

	// Build RabbitMQ URL
	config.RabbitMQ.URL = "amqp://" + config.RabbitMQ.User + ":" +
		config.RabbitMQ.Password + "@" + config.RabbitMQ.Host +
		":" + strconv.Itoa(config.RabbitMQ.Port) + config.RabbitMQ.VHost

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
