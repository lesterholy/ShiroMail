package bootstrap

import (
	"context"
	"testing"
	"time"

	"shiro-email/backend/internal/modules/ingest"
	"shiro-email/backend/internal/modules/mailbox"
	"shiro-email/backend/internal/modules/message"
)

func TestRunWorkerCycleSkipsLegacySyncAndStillCleansExpiredMailboxes(t *testing.T) {
	mailboxRepo := mailbox.NewMemoryRepository()
	messageRepo := message.NewMemoryRepository()

	expiredMailbox, err := mailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    1,
		DomainID:  1,
		Domain:    "example.test",
		LocalPart: "expired",
		Address:   "expired@example.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create expired mailbox: %v", err)
	}

	if err := messageRepo.UpsertFromLegacySync(context.Background(), expiredMailbox.ID, expiredMailbox.LocalPart, ingest.ParsedMessage{
		LegacyMailboxKey: expiredMailbox.LocalPart,
		LegacyMessageKey: "expired-1",
		FromAddr:         "cleanup@example.com",
		ToAddr:           expiredMailbox.Address,
		Subject:          "Cleanup me",
		TextPreview:      "cleanup-body",
		HTMLPreview:      "<p>cleanup-body</p>",
		ReceivedAt:       time.Now(),
	}); err != nil {
		t.Fatalf("seed legacy sync message: %v", err)
	}

	if err := runWorkerCycle(context.Background(), mailboxRepo, messageRepo, nil); err != nil {
		t.Fatalf("expected cleanup-only worker cycle to succeed, got %v", err)
	}

	msg, err := messageRepo.GetByMailboxAndID(context.Background(), expiredMailbox.ID, 1)
	if err != nil {
		t.Fatalf("load cleaned message: %v", err)
	}
	if !msg.IsDeleted {
		t.Fatalf("expected message soft-deleted after worker cycle: %+v", msg)
	}
}
