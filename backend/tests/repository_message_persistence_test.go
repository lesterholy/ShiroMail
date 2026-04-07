package tests

import (
	"context"
	"testing"
	"time"

	"shiro-email/backend/internal/modules/ingest"
	"shiro-email/backend/internal/modules/message"
)

func TestMySQLMessageRepositoryDeduplicatesByLegacyMessageKey(t *testing.T) {
	db := mustOpenTestMySQL(t)
	ctx := context.Background()

	mustResetPersistenceTables(t, db)
	mustEnsureSchema(t, db)
	mailboxID := mustSeedMailbox(t, db, mustSeedUser(t, db, "msg-user", "msg-user@example.com", []string{"user"}), mustSeedDomain(t, db, "mail.persist.test"), "seed")
	repo := message.NewMySQLRepository(db)

	parsed := ingest.ParsedMessage{
		LegacyMailboxKey: "seed",
		LegacyMessageKey: "dup-1",
		FromAddr:         "hello@example.com",
		ToAddr:           "seed@mail.persist.test",
		Subject:          "Hello",
		TextPreview:      "first",
		HTMLPreview:      "<p>first</p>",
		ReceivedAt:       time.Now(),
	}

	if err := repo.UpsertFromLegacySync(ctx, mailboxID, "seed", parsed); err != nil {
		t.Fatalf("expected first upsert success, got %v", err)
	}
	parsed.TextPreview = "second"
	if err := repo.UpsertFromLegacySync(ctx, mailboxID, "seed", parsed); err != nil {
		t.Fatalf("expected second upsert success, got %v", err)
	}

	items, err := repo.ListByMailboxID(ctx, mailboxID)
	if err != nil {
		t.Fatalf("expected list success, got %v", err)
	}
	if len(items) != 1 || items[0].TextPreview != "second" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestMySQLMessageRepositoryStoresInboundPayload(t *testing.T) {
	db := mustOpenTestMySQL(t)
	ctx := context.Background()

	mustResetPersistenceTables(t, db)
	mustEnsureSchema(t, db)
	mailboxID := mustSeedMailbox(t, db, mustSeedUser(t, db, "smtp-user", "smtp-user@example.com", []string{"user"}), mustSeedDomain(t, db, "smtp.persist.test"), "alpha")
	repo := message.NewMySQLRepository(db)

	receivedAt := time.Date(2026, 4, 2, 9, 30, 0, 0, time.UTC)
	err := repo.StoreInbound(ctx, mailboxID, ingest.StoredInboundMessage{
		SourceKind:      "smtp",
		SourceMessageID: "smtp-1",
		MailboxAddress:  "alpha@smtp.persist.test",
		FromAddr:        "sender@example.com",
		ToAddr:          "alpha@smtp.persist.test",
		Subject:         "Welcome",
		TextPreview:     "hello",
		HTMLPreview:     "<p>hello</p>",
		TextBody:        "hello",
		HTMLBody:        "<p>hello</p>",
		Headers:         map[string][]string{"Subject": {"Welcome"}},
		RawStorageKey:   "mail/2026/04/02/smtp-1.eml",
		HasAttachments:  true,
		ReceivedAt:      receivedAt,
		Attachments: []ingest.StoredAttachment{
			{
				FileName:    "note.txt",
				ContentType: "text/plain",
				StorageKey:  "attachments/smtp-1/note.txt",
				SizeBytes:   15,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected store inbound success, got %v", err)
	}

	items, err := repo.ListByMailboxID(ctx, mailboxID)
	if err != nil {
		t.Fatalf("expected list success, got %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 message, got %d", len(items))
	}
	if items[0].SourceKind != "smtp" {
		t.Fatalf("expected source kind smtp, got %s", items[0].SourceKind)
	}
	if items[0].SourceMessageID != "smtp-1" {
		t.Fatalf("expected source message id smtp-1, got %s", items[0].SourceMessageID)
	}
	if items[0].MailboxAddress != "alpha@smtp.persist.test" {
		t.Fatalf("expected mailbox address alpha@smtp.persist.test, got %s", items[0].MailboxAddress)
	}
	if items[0].RawStorageKey != "mail/2026/04/02/smtp-1.eml" {
		t.Fatalf("expected raw storage key, got %s", items[0].RawStorageKey)
	}
	if !items[0].HasAttachments {
		t.Fatal("expected hasAttachments to be true")
	}
	if items[0].Headers["Subject"][0] != "Welcome" {
		t.Fatalf("expected Subject header, got %+v", items[0].Headers)
	}
	if len(items[0].Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(items[0].Attachments))
	}
	if items[0].LegacyMailboxKey != "alpha@smtp.persist.test" {
		t.Fatalf("expected compatibility mailbox key to be populated, got %q", items[0].LegacyMailboxKey)
	}
	if items[0].LegacyMessageKey != "smtp-1" {
		t.Fatalf("expected compatibility message key to be populated, got %q", items[0].LegacyMessageKey)
	}
}

func TestMySQLMessageRepositoryDeduplicatesInboundPayloadByMailboxAndSourceMessageID(t *testing.T) {
	db := mustOpenTestMySQL(t)
	ctx := context.Background()

	mustResetPersistenceTables(t, db)
	mustEnsureSchema(t, db)
	mailboxID := mustSeedMailbox(t, db, mustSeedUser(t, db, "smtp-dedupe", "smtp-dedupe@example.com", []string{"user"}), mustSeedDomain(t, db, "smtp.dedupe.test"), "alpha")
	repo := message.NewMySQLRepository(db)

	firstReceivedAt := time.Date(2026, 4, 2, 9, 30, 0, 0, time.UTC)
	err := repo.StoreInbound(ctx, mailboxID, ingest.StoredInboundMessage{
		SourceKind:      "smtp",
		SourceMessageID: "smtp-dup-1",
		MailboxAddress:  "alpha@smtp.dedupe.test",
		FromAddr:        "sender@example.com",
		ToAddr:          "alpha@smtp.dedupe.test",
		Subject:         "First subject",
		TextPreview:     "first",
		TextBody:        "first",
		Headers:         map[string][]string{"Subject": {"First subject"}},
		RawStorageKey:   "mail/raw-first.eml",
		ReceivedAt:      firstReceivedAt,
	})
	if err != nil {
		t.Fatalf("expected first inbound store success, got %v", err)
	}

	secondReceivedAt := firstReceivedAt.Add(5 * time.Minute)
	err = repo.StoreInbound(ctx, mailboxID, ingest.StoredInboundMessage{
		SourceKind:      "smtp",
		SourceMessageID: "smtp-dup-1",
		MailboxAddress:  "alpha@smtp.dedupe.test",
		FromAddr:        "sender@example.com",
		ToAddr:          "alpha@smtp.dedupe.test",
		Subject:         "Updated subject",
		TextPreview:     "second",
		TextBody:        "second",
		Headers:         map[string][]string{"Subject": {"Updated subject"}},
		RawStorageKey:   "mail/raw-second.eml",
		ReceivedAt:      secondReceivedAt,
		Attachments: []ingest.StoredAttachment{
			{
				FileName:    "note.txt",
				ContentType: "text/plain",
				StorageKey:  "attachments/smtp-dup-1/note.txt",
				SizeBytes:   15,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected second inbound store success, got %v", err)
	}

	items, err := repo.ListByMailboxID(ctx, mailboxID)
	if err != nil {
		t.Fatalf("expected list success, got %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 message after dedupe, got %d", len(items))
	}
	if items[0].Subject != "Updated subject" {
		t.Fatalf("expected updated subject, got %q", items[0].Subject)
	}
	if items[0].RawStorageKey != "mail/raw-second.eml" {
		t.Fatalf("expected updated raw storage key, got %q", items[0].RawStorageKey)
	}
	if len(items[0].Attachments) != 1 {
		t.Fatalf("expected 1 attachment after dedupe, got %d", len(items[0].Attachments))
	}
}
