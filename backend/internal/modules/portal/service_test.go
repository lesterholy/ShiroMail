package portal

import (
	"context"
	"testing"

	"shiro-email/backend/internal/modules/auth"
	"shiro-email/backend/internal/shared/security"
)

func TestServiceCreatesOnlyMinimalDefaultsForNewUser(t *testing.T) {
	ctx := context.Background()
	authRepo := auth.NewMemoryRepository()
	repo := NewMemoryRepository()
	service := NewService(repo, authRepo)

	passwordHash, err := security.HashPassword("Secret123!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user, err := authRepo.CreateUser(ctx, auth.User{
		Username:      "clean-user",
		Email:         "clean-user@example.com",
		PasswordHash:  passwordHash,
		EmailVerified: true,
		Roles:         []string{"user"},
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	settings, err := service.GetSettings(ctx, user.ID)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if settings.DisplayName != "clean-user" {
		t.Fatalf("expected clean display name, got %+v", settings)
	}

	billing, err := service.GetBilling(ctx, user.ID)
	if err != nil {
		t.Fatalf("get billing: %v", err)
	}
	if billing.PlanCode != "free" || billing.MailboxQuota != 3 || billing.DomainQuota != 1 {
		t.Fatalf("expected minimal free billing defaults, got %+v", billing)
	}

	notices, err := service.ListNotices(ctx)
	if err != nil {
		t.Fatalf("list notices: %v", err)
	}
	if len(notices) != 0 {
		t.Fatalf("expected no default notices, got %+v", notices)
	}

	docs, err := service.ListDocs(ctx)
	if err != nil {
		t.Fatalf("list docs: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("expected no default docs, got %+v", docs)
	}

	apiKeys, err := service.ListAPIKeys(ctx, user.ID)
	if err != nil {
		t.Fatalf("list api keys: %v", err)
	}
	if len(apiKeys) != 0 {
		t.Fatalf("expected no default api keys, got %+v", apiKeys)
	}

	webhooks, err := service.ListWebhooks(ctx, user.ID)
	if err != nil {
		t.Fatalf("list webhooks: %v", err)
	}
	if len(webhooks) != 0 {
		t.Fatalf("expected no default webhooks, got %+v", webhooks)
	}

	balance, err := service.GetBalance(ctx, user.ID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if balance["balanceCents"] != int64(0) {
		t.Fatalf("expected zero default balance, got %+v", balance)
	}
	entries, ok := balance["entries"].([]BalanceEntry)
	if !ok {
		t.Fatalf("expected typed balance entries, got %#v", balance["entries"])
	}
	if len(entries) != 0 {
		t.Fatalf("expected no default balance entries, got %+v", entries)
	}
}
