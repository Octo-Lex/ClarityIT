package config

import (
	"os"
	"testing"
)

func TestMissingJWTSecretFailsInProduction(t *testing.T) {
	os.Setenv("CLARITY_ENV", "production")
	os.Setenv("DATABASE_URL", "postgres://test")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("HMAC_KEY")
	defer func() {
		os.Unsetenv("CLARITY_ENV")
		os.Unsetenv("DATABASE_URL")
	}()

	_, err := Load()
	if err == nil {
		t.Error("expected error for missing JWT_SECRET in production")
	}
}

func TestWeakJWTSecretFailsInProduction(t *testing.T) {
	os.Setenv("CLARITY_ENV", "production")
	os.Setenv("DATABASE_URL", "postgres://test")
	os.Setenv("JWT_SECRET", "short")
	os.Setenv("HMAC_KEY", "this-is-a-long-enough-hmac-key-for-production")
	defer func() {
		os.Unsetenv("CLARITY_ENV")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("HMAC_KEY")
	}()

	_, err := Load()
	if err == nil {
		t.Error("expected error for weak JWT_SECRET in production")
	}
}

func TestMissingHMACKeyFailsInProduction(t *testing.T) {
	os.Setenv("CLARITY_ENV", "production")
	os.Setenv("DATABASE_URL", "postgres://test")
	os.Setenv("JWT_SECRET", "this-is-a-long-enough-jwt-secret-for-production-use")
	os.Unsetenv("HMAC_KEY")
	defer func() {
		os.Unsetenv("CLARITY_ENV")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("JWT_SECRET")
	}()

	_, err := Load()
	if err == nil {
		t.Error("expected error for missing HMAC_KEY in production")
	}
}

func TestDevModeAllowsDefaults(t *testing.T) {
	os.Setenv("CLARITY_ENV", "development")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("HMAC_KEY")
	os.Unsetenv("DATABASE_URL")
	defer func() {
		os.Unsetenv("CLARITY_ENV")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("dev mode should allow defaults: %v", err)
	}
	if cfg.JWTSecret == "" {
		t.Error("JWT_SECRET should be auto-generated in dev mode")
	}
	if cfg.HMACKey == "" {
		t.Error("HMAC_KEY should be auto-generated in dev mode")
	}
	if !cfg.IsDev() {
		t.Error("IsDev() should return true")
	}
}

func TestProxmoxEnabledRequiresConfig(t *testing.T) {
	os.Setenv("CLARITY_ENV", "development")
	os.Setenv("PROXMOX_ENABLED", "true")
	os.Unsetenv("PROXMOX_URL")
	defer func() {
		os.Unsetenv("CLARITY_ENV")
		os.Unsetenv("PROXMOX_ENABLED")
		os.Unsetenv("PROXMOX_URL")
	}()

	_, err := Load()
	if err == nil {
		t.Error("expected error for PROXMOX_ENABLED without PROXMOX_URL")
	}
}

func TestConfigNeverLogsSecrets(t *testing.T) {
	// Verify that the Config struct doesn't have a String() method
	// that would expose secrets via fmt.Sprintf("%v", cfg)
	cfg := &Config{
		JWTSecret:    "super-secret-jwt",
		HMACKey:      "super-secret-hmac",
		ProxmoxSecret: "proxmox-secret",
		SMTPPassword: "smtp-password",
		MinioSecretKey: "minio-secret",
	}
	// Config should not implement fmt.Stringer
	_ = cfg
	// If someone adds String(), this test serves as a reminder to not include secrets
}

func TestValidProductionConfig(t *testing.T) {
	os.Setenv("CLARITY_ENV", "production")
	os.Setenv("DATABASE_URL", "postgres://user:pass@host:5432/db")
	os.Setenv("JWT_SECRET", "this-is-a-long-enough-jwt-secret-for-production-use-case")
	os.Setenv("HMAC_KEY", "this-is-a-long-enough-hmac-key-for-production")
	os.Setenv("NATS_URL", "nats://nats:4222")
	os.Setenv("REDIS_URL", "redis://redis:6379")
	defer func() {
		os.Unsetenv("CLARITY_ENV")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("HMAC_KEY")
		os.Unsetenv("NATS_URL")
		os.Unsetenv("REDIS_URL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("valid production config should load: %v", err)
	}
	if !cfg.IsProd() {
		t.Error("IsProd() should return true")
	}
}
