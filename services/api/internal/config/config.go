package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL     string
	NATSURL         string
	RedisURL        string
	JWTSecret       string
	HMACKey         string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Port            string
	ProxmoxURL      string
	ProxmoxTokenID  string
	ProxmoxSecret   string
	ProxmoxVerifyTLS bool
	MinioEndpoint   string
	MinioAccessKey  string
	MinioSecretKey  string
	MinioBucket     string
	MinioUseSSL     bool
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"),
		NATSURL:         getEnv("NATS_URL", "nats://nats:4222"),
		RedisURL:        getEnv("REDIS_URL", "redis://redis:6379"),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		HMACKey:         getEnv("HMAC_KEY", "clarityit-dev-hmac-key-change-in-production"),
		AccessTokenTTL:  getEnvDuration("ACCESS_TOKEN_TTL_MINUTES", 15) * time.Minute,
		RefreshTokenTTL: getEnvDuration("REFRESH_TOKEN_TTL_DAYS", 7) * 24 * time.Hour,
		Port:            getEnv("PORT", "8765"),
		ProxmoxURL:      getEnv("PROXMOX_URL", ""),
		ProxmoxTokenID:  getEnv("PROXMOX_TOKEN_ID", ""),
		ProxmoxSecret:   getEnv("PROXMOX_TOKEN_SECRET", ""),
		ProxmoxVerifyTLS: getEnvBool("PROXMOX_VERIFY_TLS", false),
		MinioEndpoint:   getEnv("MINIO_ENDPOINT", "minio:9000"),
		MinioAccessKey:  getEnv("MINIO_ROOT_USER", "clarityit"),
		MinioSecretKey:  getEnv("MINIO_ROOT_PASSWORD", "clarityit123"),
		MinioBucket:     getEnv("MINIO_BUCKET", "clarityit"),
		MinioUseSSL:     getEnvBool("MINIO_USE_SSL", false),
	}

	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "clarityit-dev-jwt-secret-change-in-production"
	}

	return cfg, nil
}

func (c *Config) RedisURLHost() string {
	s := strings.TrimPrefix(c.RedisURL, "redis://")
	if idx := strings.Index(s, "/"); idx >= 0 {
		s = s[:idx]
	}
	return s
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback int) time.Duration {
	return time.Duration(getEnvInt(key, fallback))
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" { return fallback }
	n, _ := strconv.Atoi(v)
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" { return fallback }
	b, _ := strconv.ParseBool(v)
	return b
}
