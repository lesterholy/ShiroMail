package ingest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"shiro-email/backend/internal/modules/mailbox"
)

func TestDirectServiceDeliversInboundMailToMatchedMailbox(t *testing.T) {
	ctx := context.Background()
	mailboxes := mailbox.NewMemoryRepository()
	target, err := mailboxes.Create(ctx, mailbox.Mailbox{
		UserID:    1,
		DomainID:  1,
		Domain:    "example.test",
		LocalPart: "alpha",
		Address:   "alpha@example.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(2 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create mailbox: %v", err)
	}

	messageStore := NewMemoryMessageRepository()
	storage, err := NewLocalFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("create local file storage: %v", err)
	}

	service := NewDirectService(mailboxes, messageStore, storage)
	result, err := service.Deliver(ctx, InboundEnvelope{
		MailFrom:   "sender@example.com",
		Recipients: []string{"alpha@example.test"},
	}, strings.NewReader("From: sender@example.com\r\nTo: alpha@example.test\r\nSubject: Welcome\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=abc\r\n\r\n--abc\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nhello direct smtp\r\n--abc\r\nContent-Type: text/plain\r\nContent-Disposition: attachment; filename=\"note.txt\"\r\n\r\nattachment body\r\n--abc--\r\n"))
	if err != nil {
		t.Fatalf("deliver inbound message: %v", err)
	}
	if result.MailboxAddress != "alpha@example.test" {
		t.Fatalf("expected alpha@example.test, got %s", result.MailboxAddress)
	}
	if result.SourceKind != "smtp" {
		t.Fatalf("expected smtp source kind, got %s", result.SourceKind)
	}
	if result.RawStorageKey == "" {
		t.Fatal("expected raw storage key to be set")
	}
	if len(result.Attachments) != 1 {
		t.Fatalf("expected 1 stored attachment, got %d", len(result.Attachments))
	}

	items, err := messageStore.ListByMailboxID(ctx, target.ID)
	if err != nil {
		t.Fatalf("list stored messages: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 stored message, got %d", len(items))
	}
	if items[0].Subject != "Welcome" {
		t.Fatalf("expected Welcome subject, got %s", items[0].Subject)
	}
	if _, err := os.Stat(filepath.Join(storage.rootDir, filepath.FromSlash(result.RawStorageKey))); err != nil {
		t.Fatalf("expected raw message file to exist, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(storage.rootDir, filepath.FromSlash(result.Attachments[0].StorageKey))); err != nil {
		t.Fatalf("expected attachment file to exist, got %v", err)
	}
}

func TestDirectServiceDeliversToMultipleMailboxesSequentially(t *testing.T) {
	ctx := context.Background()
	mailboxes := mailbox.NewMemoryRepository()
	first, err := mailboxes.Create(ctx, mailbox.Mailbox{
		UserID:    1,
		DomainID:  1,
		Domain:    "example.test",
		LocalPart: "alpha",
		Address:   "alpha@example.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(2 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create first mailbox: %v", err)
	}
	second, err := mailboxes.Create(ctx, mailbox.Mailbox{
		UserID:    1,
		DomainID:  1,
		Domain:    "example.test",
		LocalPart: "beta",
		Address:   "beta@example.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(2 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create second mailbox: %v", err)
	}

	messageStore := NewMemoryMessageRepository()
	storage, err := NewLocalFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("create local file storage: %v", err)
	}

	service := NewDirectService(mailboxes, messageStore, storage)
	deliver := func(recipient string, subject string) {
		t.Helper()
		_, deliverErr := service.Deliver(ctx, InboundEnvelope{
			MailFrom:   "sender@example.com",
			Recipients: []string{recipient},
		}, strings.NewReader("From: sender@example.com\r\nTo: "+recipient+"\r\nSubject: "+subject+"\r\n\r\nbody"))
		if deliverErr != nil {
			t.Fatalf("deliver to %s: %v", recipient, deliverErr)
		}
	}

	deliver(first.Address, "first")
	deliver(second.Address, "second")

	firstItems, err := messageStore.ListByMailboxID(ctx, first.ID)
	if err != nil {
		t.Fatalf("list first mailbox messages: %v", err)
	}
	if len(firstItems) != 1 || firstItems[0].Subject != "first" {
		t.Fatalf("expected first mailbox to receive first message, got %#v", firstItems)
	}

	secondItems, err := messageStore.ListByMailboxID(ctx, second.ID)
	if err != nil {
		t.Fatalf("list second mailbox messages: %v", err)
	}
	if len(secondItems) != 1 || secondItems[0].Subject != "second" {
		t.Fatalf("expected second mailbox to receive second message, got %#v", secondItems)
	}
}

func TestDirectServiceDeliversSingleMessageToMultipleRecipients(t *testing.T) {
	ctx := context.Background()
	mailboxes := mailbox.NewMemoryRepository()
	first, err := mailboxes.Create(ctx, mailbox.Mailbox{
		UserID:    1,
		DomainID:  1,
		Domain:    "example.test",
		LocalPart: "alpha",
		Address:   "alpha@example.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(2 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create first mailbox: %v", err)
	}
	second, err := mailboxes.Create(ctx, mailbox.Mailbox{
		UserID:    2,
		DomainID:  1,
		Domain:    "example.test",
		LocalPart: "beta",
		Address:   "beta@example.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(2 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create second mailbox: %v", err)
	}

	messageStore := NewMemoryMessageRepository()
	storage, err := NewLocalFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("create local file storage: %v", err)
	}

	service := NewDirectService(mailboxes, messageStore, storage)
	_, err = service.Deliver(ctx, InboundEnvelope{
		MailFrom:   "sender@example.com",
		Recipients: []string{first.Address, second.Address},
	}, strings.NewReader("From: sender@example.com\r\nTo: alpha@example.test, beta@example.test\r\nSubject: broadcast\r\n\r\nbody"))
	if err != nil {
		t.Fatalf("deliver to multiple recipients: %v", err)
	}

	firstItems, err := messageStore.ListByMailboxID(ctx, first.ID)
	if err != nil {
		t.Fatalf("list first mailbox messages: %v", err)
	}
	if len(firstItems) != 1 || firstItems[0].Subject != "broadcast" {
		t.Fatalf("expected first mailbox broadcast, got %#v", firstItems)
	}

	secondItems, err := messageStore.ListByMailboxID(ctx, second.ID)
	if err != nil {
		t.Fatalf("list second mailbox messages: %v", err)
	}
	if len(secondItems) != 1 || secondItems[0].Subject != "broadcast" {
		t.Fatalf("expected second mailbox broadcast, got %#v", secondItems)
	}
}

func TestBuildPreviewKeepsUTF8Boundary(t *testing.T) {
	body := strings.Repeat("测", 200)

	preview := buildPreview(body)

	if preview != strings.Repeat("测", 160) {
		t.Fatalf("expected 160 full runes, got %q", preview)
	}
	if !utf8.ValidString(preview) {
		t.Fatal("expected preview to remain valid utf-8")
	}
}
