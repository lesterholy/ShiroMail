package config

import "testing"

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("APP_PORT", "8080")
	t.Setenv("CLOUDFLARE_API_BASE_URL", "https://cf.test/client/v4")
	t.Setenv("SPACESHIP_API_BASE_URL", "https://spaceship.test/api")
	cfg := MustLoadConfig()
	if cfg.AppPort != "8080" {
		t.Fatalf("expected app port 8080, got %s", cfg.AppPort)
	}
	if cfg.CloudflareAPIBaseURL != "https://cf.test/client/v4" {
		t.Fatalf("expected cloudflare api base url override, got %s", cfg.CloudflareAPIBaseURL)
	}
	if cfg.SpaceshipAPIBaseURL != "https://spaceship.test/api" {
		t.Fatalf("expected spaceship api base url override, got %s", cfg.SpaceshipAPIBaseURL)
	}
	t.Setenv("LEGACY_MAIL_SYNC_ENABLED", "")
	if cfg.LegacyMailSyncEnabled {
		t.Fatal("expected legacy mail sync to be disabled by default")
	}
}
