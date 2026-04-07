package bootstrap

import (
	"context"
	"testing"

	ingestsmtp "shiro-email/backend/internal/modules/ingest/smtp"
	"shiro-email/backend/internal/modules/system"
)

func TestResolveSMTPRuntimeSettingsUsesStoredSettingsWhenPresent(t *testing.T) {
	repo := system.NewMemoryConfigRepository()
	_, err := repo.Upsert(context.Background(), system.ConfigKeyMailSMTP, map[string]any{
		"enabled":         true,
		"listenAddr":      "0.0.0.0:2626",
		"hostname":        "smtp.galiais.com",
		"dkimCnameTarget": "dkim._domainkey.galiais.com",
		"maxMessageBytes": 20971520,
	}, 1)
	if err != nil {
		t.Fatalf("seed mail smtp config: %v", err)
	}

	enabled, resolved := resolveSMTPRuntimeSettings(context.Background(), repo)
	if !enabled {
		t.Fatal("expected smtp to stay enabled")
	}

	expected := ingestsmtp.Config{
		ListenAddr:      "0.0.0.0:2626",
		Hostname:        "smtp.galiais.com",
		MaxMessageBytes: 20971520,
	}
	if resolved != expected {
		t.Fatalf("expected stored smtp config %+v, got %+v", expected, resolved)
	}
}

func TestResolveSMTPRuntimeSettingsFallsBackToDefaultsWithoutStoredSettings(t *testing.T) {
	repo := system.NewMemoryConfigRepository()

	enabled, resolved := resolveSMTPRuntimeSettings(context.Background(), repo)
	if !enabled {
		t.Fatal("expected smtp defaults to stay enabled")
	}

	expected := ingestsmtp.Config{
		ListenAddr:      ":2525",
		Hostname:        "mail.shiro.local",
		MaxMessageBytes: 10485760,
	}
	if resolved != expected {
		t.Fatalf("expected default smtp config %+v, got %+v", expected, resolved)
	}
}
