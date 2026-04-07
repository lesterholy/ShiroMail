package system

import (
	"context"
	"testing"
)

type stubMailDeliveryTester struct {
	lastTo string
}

func (s *stubMailDeliveryTester) SendTestMail(_ context.Context, to string) error {
	s.lastTo = to
	return nil
}

func TestSendMailDeliveryTestUsesConfiguredFromAddressByDefault(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	tester := &stubMailDeliveryTester{}

	_, _ = configRepo.Upsert(context.Background(), ConfigKeyMailDelivery, map[string]any{
		"enabled":     true,
		"host":        "smtp.example.com",
		"port":        587,
		"username":    "sender@example.com",
		"password":    "app-password",
		"fromAddress": "sender@example.com",
		"fromName":    "Shiro Email",
	}, 1)

	service := NewService(configRepo, jobRepo, auditRepo, tester)

	result, err := service.SendMailDeliveryTest(context.Background(), 99, "")
	if err != nil {
		t.Fatalf("send test mail: %v", err)
	}
	if result.Recipient != "sender@example.com" {
		t.Fatalf("expected fallback recipient, got %q", result.Recipient)
	}
	if tester.lastTo != "sender@example.com" {
		t.Fatalf("expected tester recipient sender@example.com, got %q", tester.lastTo)
	}

	audits, err := auditRepo.List(context.Background())
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(audits) == 0 || audits[0].Action != "admin.mail_delivery.test" {
		t.Fatalf("expected audit log for mail delivery test, got %+v", audits)
	}
}

func TestListSettingsSectionsIncludesDynamicOAuthProviders(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	service := NewService(configRepo, jobRepo, auditRepo)

	_, _ = configRepo.Upsert(context.Background(), "auth.oauth.providers.discord", map[string]any{
		"displayName": "Discord",
		"enabled":     true,
	}, 1)

	sections, err := service.ListSettingsSections(context.Background())
	if err != nil {
		t.Fatalf("list settings sections: %v", err)
	}

	var oauthSection *SettingsSection
	for index := range sections {
		if sections[index].Key == "oauth" {
			oauthSection = &sections[index]
			break
		}
	}
	if oauthSection == nil {
		t.Fatal("expected oauth section to exist")
	}

	foundDisplay := false
	foundDiscord := false
	for _, item := range oauthSection.Items {
		if item.Key == ConfigKeyAuthOAuthDisplay {
			foundDisplay = true
		}
		if item.Key == "auth.oauth.providers.discord" {
			foundDiscord = true
		}
	}

	if !foundDisplay {
		t.Fatal("expected oauth section to include display config")
	}
	if !foundDiscord {
		t.Fatal("expected oauth section to include dynamic provider config")
	}
}

func TestListSettingsSectionsDoesNotRestoreDeletedDefaultOAuthProviders(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	service := NewService(configRepo, jobRepo, auditRepo)

	sections, err := service.ListSettingsSections(context.Background())
	if err != nil {
		t.Fatalf("list settings sections: %v", err)
	}

	var oauthSection *SettingsSection
	for index := range sections {
		if sections[index].Key == "oauth" {
			oauthSection = &sections[index]
			break
		}
	}
	if oauthSection == nil {
		t.Fatal("expected oauth section to exist")
	}

	if len(oauthSection.Items) != 1 {
		t.Fatalf("expected only oauth display config by default, got %d items", len(oauthSection.Items))
	}
	if oauthSection.Items[0].Key != ConfigKeyAuthOAuthDisplay {
		t.Fatalf("expected oauth display config only, got %q", oauthSection.Items[0].Key)
	}
}

func TestLoadAuthRuntimeSettingsDoesNotInjectDefaultOAuthProvidersFromEmptyRepo(t *testing.T) {
	configRepo := NewMemoryConfigRepository()

	settings, err := LoadAuthRuntimeSettings(context.Background(), configRepo)
	if err != nil {
		t.Fatalf("load auth runtime settings: %v", err)
	}

	if len(settings.OAuth) != 0 {
		t.Fatalf("expected no oauth providers from empty repo, got %+v", settings.OAuth)
	}
}
