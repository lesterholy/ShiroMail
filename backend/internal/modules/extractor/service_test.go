package extractor

import (
	"context"
	"testing"
	"time"

	"shiro-email/backend/internal/modules/mailbox"
)

func TestServiceExtractsVerificationCodeFromSubject(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)

	rule, err := service.CreatePortalRule(context.Background(), 7, UpsertRuleInput{
		Name:              "subject code",
		Label:             "验证码",
		Enabled:           true,
		TargetFields:      []TargetField{TargetFieldSubject},
		Pattern:           `\b(\d{6})\b`,
		ResultMode:        ResultModeCaptureGroup,
		CaptureGroupIndex: intPtr(1),
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	result, err := extractForRules([]Rule{rule}, MessageContent{
		MailboxID: 3,
		DomainID:  5,
		Subject:   "Your code is 741612",
	})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 match, got %#v", result.Items)
	}
	if result.Items[0].Value != "741612" {
		t.Fatalf("expected code 741612, got %#v", result.Items[0])
	}
}

func TestServiceHonorsMailboxScope(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)

	rule, err := service.CreatePortalRule(context.Background(), 7, UpsertRuleInput{
		Name:         "scoped link",
		Label:        "重置链接",
		Enabled:      true,
		TargetFields: []TargetField{TargetFieldTextBody},
		Pattern:      `https://[^\s]+`,
		ResultMode:   ResultModeFirstMatch,
		MailboxIDs:   []uint64{11},
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	result, err := extractForRules([]Rule{rule}, MessageContent{
		MailboxID: 12,
		DomainID:  5,
		TextBody:  "reset https://example.com/reset",
	})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected no matches outside scope, got %#v", result.Items)
	}
}

func TestServiceRejectsInvalidRegex(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)

	_, err := service.CreatePortalRule(context.Background(), 7, UpsertRuleInput{
		Name:         "bad",
		Enabled:      true,
		TargetFields: []TargetField{TargetFieldSubject},
		Pattern:      "(",
		ResultMode:   ResultModeFirstMatch,
	})
	if err == nil {
		t.Fatal("expected invalid regex error")
	}
}

func TestServiceListsEnabledAdminTemplatesSeparately(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)

	template, err := service.CreateAdminRule(context.Background(), 1, UpsertRuleInput{
		Name:              "验证码模板",
		Label:             "验证码",
		Enabled:           true,
		TargetFields:      []TargetField{TargetFieldSubject},
		Pattern:           `\b(\d{6})\b`,
		ResultMode:        ResultModeCaptureGroup,
		CaptureGroupIndex: intPtr(1),
	})
	if err != nil {
		t.Fatalf("create admin template: %v", err)
	}
	if err := service.EnableTemplate(context.Background(), 9, template.ID); err != nil {
		t.Fatalf("enable template: %v", err)
	}

	list, err := service.ListPortalRules(context.Background(), 9)
	if err != nil {
		t.Fatalf("list portal rules: %v", err)
	}
	if len(list.Templates) != 1 || !list.Templates[0].EnabledForUser {
		t.Fatalf("expected enabled template in list, got %#v", list.Templates)
	}
}

func TestServiceDropsDeletedMailboxScopeOnCreateAndList(t *testing.T) {
	repo := NewMemoryRepository()
	mailboxRepo := mailbox.NewMemoryRepository()
	service := NewService(repo, mailboxRepo, nil, nil)

	activeBox, err := mailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    7,
		DomainID:  1,
		Domain:    "scope.test",
		LocalPart: "active",
		Address:   "active@scope.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create active mailbox: %v", err)
	}
	deletedBox, err := mailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    7,
		DomainID:  1,
		Domain:    "scope.test",
		LocalPart: "deleted",
		Address:   "deleted@scope.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create deleted mailbox fixture: %v", err)
	}
	if err := mailboxRepo.DeleteByID(context.Background(), deletedBox.ID); err != nil {
		t.Fatalf("delete mailbox fixture: %v", err)
	}

	rule, err := service.CreatePortalRule(context.Background(), 7, UpsertRuleInput{
		Name:         "scoped rule",
		Enabled:      true,
		TargetFields: []TargetField{TargetFieldSubject},
		Pattern:      `\b(\d{6})\b`,
		ResultMode:   ResultModeFirstMatch,
		MailboxIDs:   []uint64{activeBox.ID, deletedBox.ID},
	})
	if err != nil {
		t.Fatalf("create scoped rule: %v", err)
	}
	if len(rule.MailboxIDs) != 1 || rule.MailboxIDs[0] != activeBox.ID {
		t.Fatalf("expected deleted mailbox scope to be removed on create, got %#v", rule.MailboxIDs)
	}

	stored, err := repo.FindRuleByID(context.Background(), rule.ID)
	if err != nil {
		t.Fatalf("find stored rule: %v", err)
	}
	stored.MailboxIDs = []uint64{deletedBox.ID}
	if _, err := repo.UpdateRule(context.Background(), stored); err != nil {
		t.Fatalf("force stale mailbox scope into repo: %v", err)
	}

	list, err := service.ListPortalRules(context.Background(), 7)
	if err != nil {
		t.Fatalf("list portal rules: %v", err)
	}
	if len(list.Rules) != 1 {
		t.Fatalf("expected one rule in list, got %#v", list.Rules)
	}
	if len(list.Rules[0].MailboxIDs) != 0 {
		t.Fatalf("expected stale deleted mailbox scope to be stripped from list, got %#v", list.Rules[0].MailboxIDs)
	}
}

func intPtr(value int) *int {
	return &value
}
