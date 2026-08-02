// internal/config/config.go
package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Config содержит конфигурацию приложения
type Config struct {
	AppEnv   string
	AppPort  string
	Database DatabaseConfig
	Storage  StorageConfig
}

// DatabaseConfig содержит параметры подключения к БД
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// StorageConfig — параметры Yandex Object Storage (S3-совместимое) для загрузки медиа статей.
// Yandex Object Storage совместим с S3 API, поэтому используется обычный AWS SDK for Go
// с кастомным endpoint — отдельная интеграция не нужна.
type StorageConfig struct {
	Endpoint  string // https://storage.yandexcloud.net
	Region    string // ru-central1
	Bucket    string
	AccessKey string
	SecretKey string
	PublicURL string // базовый URL, по которому файлы отдаются публично (обычно https://<bucket>.storage.yandexcloud.net или CDN-домен)
}

// LoadConfig загружает конфигурацию из переменных окружения
func LoadConfig() *Config {
	return &Config{
		AppEnv:  getEnvOrDefault("APP_ENV", "development"),
		AppPort: getEnvOrDefault("APP_PORT", "8080"),
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     getEnvOrDefault("DB_PORT", "5432"),
			User:     getEnvOrDefault("DB_USER", "postgres"),
			Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
			Name:     getEnvOrDefault("DB_NAME", "samotsvety"),
			SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
		},
		Storage: StorageConfig{
			Endpoint:  getEnvOrDefault("YC_S3_ENDPOINT", "https://storage.yandexcloud.net"),
			Region:    getEnvOrDefault("YC_S3_REGION", "ru-central1"),
			Bucket:    getEnvOrDefault("YC_S3_BUCKET", ""),
			AccessKey: getEnvOrDefault("YC_S3_ACCESS_KEY", ""),
			SecretKey: getEnvOrDefault("YC_S3_SECRET_KEY", ""),
			PublicURL: getEnvOrDefault("YC_S3_PUBLIC_URL", ""),
		},
	}
}

// ConnectDB подключается к базе данных и проверяет соединение
func ConnectDB(ctx context.Context, cfg DatabaseConfig) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
	)

	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Проверяем соединение
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	slog.Info("Database connected", "host", cfg.Host, "name", cfg.Name)
	return db, nil
}

// GetDSN возвращает строку подключения
func (cfg DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	)
}

// Helper функции
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
