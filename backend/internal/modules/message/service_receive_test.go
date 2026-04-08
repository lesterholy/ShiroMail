package message

import (
	"context"
	"strings"
	"testing"
	"time"

	"shiro-email/backend/internal/modules/domain"
	"shiro-email/backend/internal/modules/ingest"
	"shiro-email/backend/internal/modules/mailbox"
)

func TestReceiveRawMessageReturnsQueuedPlaceholderWhenSpoolEnabled(t *testing.T) {
	ctx := context.Background()
	domainRepo := domain.NewMemoryRepository(nil)
	mailboxRepo := mailbox.NewMemoryRepository()
	messageRepo := NewMemoryRepository()
	storage, err := ingest.NewLocalFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("create local storage: %v", err)
	}

	target, err := mailboxRepo.Create(ctx, mailbox.Mailbox{
		UserID:    7,
		DomainID:  1,
		Domain:    "example.test",
		LocalPart: "queued",
		Address:   "queued@example.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create mailbox: %v", err)
	}

	receiver := ingest.NewDirectService(mailboxRepo, messageRepo, storage)
	receiver.SetSpoolRepository(ingest.NewMemorySpoolRepository())

	service := NewService(messageRepo, mailboxRepo, domainRepo, storage)
	item, err := service.ReceiveRawMessage(ctx, target.UserID, target.ID, "sender@example.com", []byte("From: sender@example.com\r\nTo: queued@example.test\r\nSubject: Queued\r\n\r\nbody"), receiver)
	if err != nil {
		t.Fatalf("receive raw message: %v", err)
	}
	if item.SourceKind != "smtp-spool" {
		t.Fatalf("expected smtp-spool placeholder, got %s", item.SourceKind)
	}
	if !strings.HasPrefix(item.SourceMessageID, "spool-") {
		t.Fatalf("expected spool source id, got %s", item.SourceMessageID)
	}
}
