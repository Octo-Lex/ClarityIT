package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment mode
const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

type Config struct {
	// Mode
	Env string // "development" or "production"

	// Core
	DatabaseURL     string
	NATSURL         string
	RedisURL        string
	JWTSecret       string
	HMACKey         string
	MFAKey          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Port            string

	// Proxmox (optional)
	ProxmoxEnabled         bool
	ProxmoxURL             string
	ProxmoxTokenID         string
	ProxmoxSecret          string
	ProxmoxVerifyTLS       bool
	ProxmoxMutationEnabled bool

	// MinIO
	MinioEndpoint   string
	MinioAccessKey  string
	MinioSecretKey  string
	MinioBucket     string
	MinioUseSSL     bool

	// Approval monitor
	ApprovalMonitorEnabled          bool
	ApprovalMonitorIntervalSeconds  int
	ApprovalExpiringThresholdPercent int

	// WebAuthn
	WebAuthnEnabled       bool
	WebAuthnRPID          string
	WebAuthnRPOrigin      string
	WebAuthnRPDisplayName string

	// SMTP (optional)
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPTLSMode  string // "none", "starttls", "tls"

	// Email
	EmailMode string // "dev", "smtp", "disabled"

	// Presenton (optional)
	PresentonEnabled           bool
	PresentonURL               string
	PresentonAdminUser         string
	PresentonAdminPass         string
	PresentonGenerationTimeout time.Duration
	PresentonMaxFileBytes      int64

	// Build info (set via ldflags)
	Version   string
	GitCommit string
}

func Load() (*Config, error) {
	env := getEnv("CLARITY_ENV", EnvDevelopment)
	cfg := &Config{
		Env: env,

		DatabaseURL:     getEnv("DATABASE_URL", ""),
		NATSURL:         getEnv("NATS_URL", ""),
		RedisURL:        getEnv("REDIS_URL", ""),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		HMACKey:         getEnv("HMAC_KEY", ""),
		MFAKey:          getEnv("MFA_KEY", ""),
		AccessTokenTTL:  getEnvDuration("ACCESS_TOKEN_TTL_MINUTES", 15) * time.Minute,
		RefreshTokenTTL: getEnvDuration("REFRESH_TOKEN_TTL_DAYS", 7) * 24 * time.Hour,
		Port:            getEnv("PORT", "8765"),

		ProxmoxEnabled:         getEnvBool("PROXMOX_ENABLED", false),
		ProxmoxURL:             getEnv("PROXMOX_URL", ""),
		ProxmoxTokenID:         getEnv("PROXMOX_TOKEN_ID", ""),
		ProxmoxSecret:          getEnv("PROXMOX_TOKEN_SECRET", ""),
		ProxmoxVerifyTLS:       getEnvBool("PROXMOX_VERIFY_TLS", false),
		ProxmoxMutationEnabled: getEnvBool("PROXMOX_MUTATION_ENABLED", false),

		MinioEndpoint:   getEnv("MINIO_ENDPOINT", ""),
		MinioAccessKey:  getEnv("MINIO_ROOT_USER", ""),
		MinioSecretKey:  getEnv("MINIO_ROOT_PASSWORD", ""),
		MinioBucket:     getEnv("MINIO_BUCKET", "clarityit"),
		MinioUseSSL:     getEnvBool("MINIO_USE_SSL", false),

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnvInt("SMTP_PORT", 587),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", ""),
		SMTPTLSMode:  getEnv("SMTP_TLS_MODE", "starttls"),
		EmailMode:    getEnv("EMAIL_MODE", "dev"),

		ApprovalMonitorEnabled:           getEnvBoolDefault("APPROVAL_MONITOR_ENABLED", env == EnvProduction),
		ApprovalMonitorIntervalSeconds:   getEnvIntClamped("APPROVAL_MONITOR_INTERVAL_SECONDS", 60, 5, 3600),
		ApprovalExpiringThresholdPercent: getEnvIntClamped("APPROVAL_EXPIRING_THRESHOLD_PERCENT", 25, 1, 90),

		WebAuthnEnabled:       getEnvBoolDefault("WEBAUTHN_ENABLED", false),
		WebAuthnRPID:          getEnv("WEBAUTHN_RP_ID", ""),
		WebAuthnRPOrigin:      getEnv("WEBAUTHN_RP_ORIGIN", ""),
		WebAuthnRPDisplayName: getEnv("WEBAUTHN_RP_DISPLAY_NAME", "ClarityIT"),

		PresentonEnabled:           getEnvBool("PRESENTON_ENABLED", false),
		PresentonURL:               getEnv("PRESENTON_URL", "http://presenton:80"),
		PresentonAdminUser:         getEnv("PRESENTON_ADMIN_USER", "clarityit"),
		PresentonAdminPass:         getEnv("PRESENTON_ADMIN_PASS", ""),
		PresentonGenerationTimeout: time.Duration(getEnvIntClamped("PRESENTON_GENERATION_TIMEOUT_SECONDS", 120, 10, 600)) * time.Second,
		PresentonMaxFileBytes:      int64(getEnvIntClamped("PRESENTON_MAX_FILE_BYTES", 52428800, 1048576, 104857600)), // 1MB-100MB, default 50MB

		Version:   getEnv("CLARITY_VERSION", "dev"),
		GitCommit: getEnv("CLARITY_GIT_COMMIT", ""),
	}

	// Apply dev defaults only in development mode
	if cfg.Env == EnvDevelopment {
		cfg.applyDevDefaults()
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// applyDevDefaults sets insecure defaults for local development only.
func (c *Config) applyDevDefaults() {
	if c.DatabaseURL == "" {
		c.DatabaseURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"
	}
	if c.NATSURL == "" {
		c.NATSURL = "nats://nats:4222"
	}
	if c.RedisURL == "" {
		c.RedisURL = "redis://redis:6379"
	}
	if c.JWTSecret == "" {
		c.JWTSecret = mustGenerateRandomKey("jwt-dev")
	}
	if c.HMACKey == "" {
		c.HMACKey = mustGenerateRandomKey("hmac-dev")
	}
	if c.MFAKey == "" {
		c.MFAKey = mustGenerateRandomKey("mfa-dev")
	}
	if c.MinioEndpoint == "" {
		c.MinioEndpoint = "minio:9000"
	}
	if c.MinioAccessKey == "" {
		c.MinioAccessKey = "clarityit"
	}
	if c.MinioSecretKey == "" {
		c.MinioSecretKey = "clarityit123"
	}
}

// Validate checks configuration for required fields and security requirements.
func (c *Config) Validate() error {
	var errs []string

	// Core required fields
	if c.DatabaseURL == "" {
		errs = append(errs, "DATABASE_URL is required")
	}
	if c.JWTSecret == "" {
		errs = append(errs, "JWT_SECRET is required")
	}
	if c.HMACKey == "" {
		errs = append(errs, "HMAC_KEY is required")
	}

	// Production-only checks
	if c.Env == EnvProduction {
		if err := validateSecretStrength("JWT_SECRET", c.JWTSecret, 32); err != nil {
			errs = append(errs, err.Error())
		}
		if err := validateSecretStrength("HMAC_KEY", c.HMACKey, 32); err != nil {
			errs = append(errs, err.Error())
		}
		if err := validateSecretStrength("MFA_KEY", c.MFAKey, 32); err != nil {
			errs = append(errs, err.Error())
		}
		// Disallow known dev defaults in production
		if c.JWTSecret == "clarityit-dev-jwt-secret-change-in-production" {
			errs = append(errs, "JWT_SECRET must be changed from dev default in production")
		}
		if c.HMACKey == "clarityit-dev-hmac-key-change-in-production" {
			errs = append(errs, "HMAC_KEY must be changed from dev default in production")
		}
		if c.MFAKey == "" || len(c.MFAKey) < 32 {
			errs = append(errs, "MFA_KEY must be at least 32 characters in production")
		}
		// NATS and Redis required in production
		if c.NATSURL == "" {
			errs = append(errs, "NATS_URL is required in production")
		}
		if c.RedisURL == "" {
			errs = append(errs, "REDIS_URL is required in production")
		}
	}

	// Proxmox validation (optional, but if enabled must have config)
	if c.ProxmoxEnabled {
		if c.ProxmoxURL == "" {
			errs = append(errs, "PROXMOX_URL is required when PROXMOX_ENABLED=true")
		}
		if c.ProxmoxTokenID == "" {
			errs = append(errs, "PROXMOX_TOKEN_ID is required when PROXMOX_ENABLED=true")
		}
		if c.ProxmoxSecret == "" {
			errs = append(errs, "PROXMOX_TOKEN_SECRET is required when PROXMOX_ENABLED=true")
		}
	}

	// SMTP validation (optional, but if host set, require credentials)
	if c.SMTPHost != "" {
		if c.SMTPUsername == "" {
			errs = append(errs, "SMTP_USERNAME is required when SMTP_HOST is set")
		}
		if c.SMTPFrom == "" {
			errs = append(errs, "SMTP_FROM is required when SMTP_HOST is set")
		}
	}

	// Email mode validation
	if c.EmailMode != "dev" && c.EmailMode != "smtp" && c.EmailMode != "disabled" {
		errs = append(errs, "EMAIL_MODE must be dev, smtp, or disabled")
	}
	if c.EmailMode == "smtp" && c.SMTPHost == "" {
		errs = append(errs, "SMTP_HOST is required when EMAIL_MODE=smtp")
	}

	// WebAuthn validation
	if c.WebAuthnEnabled {
		if c.WebAuthnRPID == "" {
			errs = append(errs, "WEBAUTHN_RP_ID is required when WEBAUTHN_ENABLED=true")
		}
		if c.WebAuthnRPOrigin == "" {
			errs = append(errs, "WEBAUTHN_RP_ORIGIN is required when WEBAUTHN_ENABLED=true")
		}
		// Production requires HTTPS origin
		if c.IsProd() {
			if !strings.HasPrefix(c.WebAuthnRPOrigin, "https://") {
				errs = append(errs, "WEBAUTHN_RP_ORIGIN must use https:// in production")
			}
		}
		// Dev allows localhost http
		if c.IsDev() {
			if !strings.HasPrefix(c.WebAuthnRPOrigin, "http://") && !strings.HasPrefix(c.WebAuthnRPOrigin, "https://") {
				errs = append(errs, "WEBAUTHN_RP_ORIGIN must start with http:// or https://")
			}
		}
	}

	// Presenton validation (optional, but if enabled must have valid config)
	if c.PresentonEnabled {
		if c.PresentonAdminPass == "" {
			errs = append(errs, "PRESENTON_ADMIN_PASS is required when PRESENTON_ENABLED=true")
		}
		if c.PresentonAdminPass == "changeme" {
			errs = append(errs, "PRESENTON_ADMIN_PASS must not be the default 'changeme'")
		}
		if c.PresentonURL == "" || (!strings.HasPrefix(c.PresentonURL, "http://") && !strings.HasPrefix(c.PresentonURL, "https://")) {
			errs = append(errs, "PRESENTON_URL must be a valid http:// or https:// URL")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// IsDev returns true if running in development mode.
func (c *Config) IsDev() bool {
	return c.Env == EnvDevelopment
}

// IsProd returns true if running in production mode.
func (c *Config) IsProd() bool {
	return c.Env == EnvProduction
}

// validateSecretStrength checks minimum entropy for production secrets.
func validateSecretStrength(name, value string, minLen int) error {
	if len(value) < minLen {
		return fmt.Errorf("%s must be at least %d characters (got %d)", name, minLen, len(value))
	}
	// Check for obvious patterns
	lower := strings.ToLower(value)
	if lower == value || strings.ToLower(value) == value {
		// All same case is weak but not fatal
	}
	return nil
}

// mustGenerateRandomKey generates a random key for dev mode.
func mustGenerateRandomKey(prefix string) string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", prefix, b)
}

func (c *Config) RedisURLHost() string {
	s := strings.TrimPrefix(c.RedisURL, "redis://")
	if idx := strings.Index(s, "/"); idx >= 0 {
		s = s[:idx]
	}
	return s
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
	if v == "" {
		return fallback
	}
	n, _ := strconv.Atoi(v)
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, _ := strconv.ParseBool(v)
	return b
}

func getEnvBoolDefault(key string, fallback bool) bool {
	return getEnvBool(key, fallback)
}

func getEnvIntClamped(key string, fallback, min, max int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
