package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"shiro-email/backend/internal/jobs"
	"shiro-email/backend/internal/modules/ingest"
	"shiro-email/backend/internal/modules/mailbox"
)

func TestSyncMailboxMessagesFromLegacyMailSync(t *testing.T) {
	repo := ingest.NewMemoryMessageRepository()
	svc := ingest.NewLegacySyncService(stubLegacyMailSyncClient{
		headersByMailbox: map[string][]ingest.LegacyMessageHeader{
			"alice": {
				{
					Mailbox: "alice",
					ID:      "msg-1",
					From:    "no-reply@example.com",
					To:      []string{"alice@example.test"},
					Subject: "Welcome",
					Date:    time.Date(2026, 4, 2, 1, 0, 0, 0, time.UTC),
					Size:    128,
				},
			},
		},
		messagesByMailbox: map[string]map[string]ingest.LegacyMessage{
			"alice": {
				"msg-1": {
					Mailbox: "alice",
					ID:      "msg-1",
					From:    "no-reply@example.com",
					To:      []string{"alice@example.test"},
					Subject: "Welcome",
					Date:    time.Date(2026, 4, 2, 1, 0, 0, 0, time.UTC),
					Size:    128,
					Body: &ingest.LegacyMessageBody{
						Text: "hello from legacy sync",
						HTML: "<p>hello from legacy sync</p>",
					},
					Attachments: []ingest.LegacyMessageAttachment{
						{
							FileName:    "hello.txt",
							ContentType: "text/plain",
						},
					},
				},
			},
		},
	}, repo)

	err := svc.SyncMailbox(context.Background(), mailboxFixture(t, 1, "alice@example.test"))
	if err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	items, err := repo.ListByMailboxID(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected stored messages, got %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 message, got %d", len(items))
	}
	if items[0].Subject != "Welcome" {
		t.Fatalf("expected subject Welcome, got %s", items[0].Subject)
	}
	if items[0].TextPreview != "hello from legacy sync" {
		t.Fatalf("expected parsed text preview, got %s", items[0].TextPreview)
	}
	if len(items[0].Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(items[0].Attachments))
	}
}

func TestSyncMailboxMessagesDeduplicatesByLegacyMessageKey(t *testing.T) {
	repo := ingest.NewMemoryMessageRepository()
	svc := ingest.NewLegacySyncService(stubLegacyMailSyncClient{
		headersByMailbox: map[string][]ingest.LegacyMessageHeader{
			"alice": {
				{
					Mailbox: "alice",
					ID:      "msg-dup",
					From:    "alerts@example.com",
					To:      []string{"alice@example.test"},
					Subject: "Alert",
					Date:    time.Date(2026, 4, 2, 2, 0, 0, 0, time.UTC),
					Size:    256,
				},
			},
		},
		messagesByMailbox: map[string]map[string]ingest.LegacyMessage{
			"alice": {
				"msg-dup": {
					Mailbox: "alice",
					ID:      "msg-dup",
					From:    "alerts@example.com",
					To:      []string{"alice@example.test"},
					Subject: "Alert",
					Date:    time.Date(2026, 4, 2, 2, 0, 0, 0, time.UTC),
					Size:    256,
					Body: &ingest.LegacyMessageBody{
						Text: "duplicate-safe",
					},
				},
			},
		},
	}, repo)

	mb := mailboxFixture(t, 7, "alice@example.test")
	if err := svc.SyncMailbox(context.Background(), mb); err != nil {
		t.Fatalf("expected first sync success, got %v", err)
	}
	if err := svc.SyncMailbox(context.Background(), mb); err != nil {
		t.Fatalf("expected second sync success, got %v", err)
	}

	items, err := repo.ListByMailboxID(context.Background(), 7)
	if err != nil {
		t.Fatalf("expected stored messages, got %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 deduplicated message, got %d", len(items))
	}
}

func TestRunSyncMessagesJobSyncsActiveMailboxes(t *testing.T) {
	mailboxRepo := mailbox.NewMemoryRepository()
	activeMailbox, err := mailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    1,
		DomainID:  1,
		Domain:    "example.test",
		LocalPart: "active",
		Address:   "active@example.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected active mailbox fixture, got %v", err)
	}
	_, err = mailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    1,
		DomainID:  1,
		Domain:    "example.test",
		LocalPart: "released",
		Address:   "released@example.test",
		Status:    "released",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected released mailbox fixture, got %v", err)
	}

	repo := ingest.NewMemoryMessageRepository()
	svc := ingest.NewLegacySyncService(stubLegacyMailSyncClient{
		headersByMailbox: map[string][]ingest.LegacyMessageHeader{
			"active": {
				{
					Mailbox: "active",
					ID:      "msg-42",
					From:    "system@example.com",
					To:      []string{"active@example.test"},
					Subject: "Job Sync",
					Date:    time.Date(2026, 4, 2, 3, 0, 0, 0, time.UTC),
					Size:    64,
				},
			},
		},
		messagesByMailbox: map[string]map[string]ingest.LegacyMessage{
			"active": {
				"msg-42": {
					Mailbox: "active",
					ID:      "msg-42",
					From:    "system@example.com",
					To:      []string{"active@example.test"},
					Subject: "Job Sync",
					Date:    time.Date(2026, 4, 2, 3, 0, 0, 0, time.UTC),
					Size:    64,
					Body: &ingest.LegacyMessageBody{
						Text: "job triggered",
					},
				},
			},
		},
	}, repo)

	err = jobs.RunSyncMessagesJob(context.Background(), mailboxRepo, svc)
	if err != nil {
		t.Fatalf("expected job sync success, got %v", err)
	}

	items, err := repo.ListByMailboxID(context.Background(), activeMailbox.ID)
	if err != nil {
		t.Fatalf("expected stored messages, got %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 active mailbox message, got %d", len(items))
	}
}

func TestRunInboundSpoolJobProcessesQueuedInboundMessage(t *testing.T) {
	ctx := context.Background()
	mailboxRepo := mailbox.NewMemoryRepository()
	target, err := mailboxRepo.Create(ctx, mailbox.Mailbox{
		UserID:    1,
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
		t.Fatalf("create queued mailbox: %v", err)
	}

	repo := ingest.NewMemoryMessageRepository()
	storage, err := ingest.NewLocalFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("create local storage: %v", err)
	}
	spool := ingest.NewMemorySpoolRepository()
	service := ingest.NewDirectService(mailboxRepo, repo, storage)
	service.SetSpoolRepository(spool)

	if _, err := service.Deliver(ctx, ingest.InboundEnvelope{
		MailFrom:   "sender@example.com",
		Recipients: []string{target.Address},
	}, strings.NewReader("From: sender@example.com\r\nTo: queued@example.test\r\nSubject: Async queued\r\n\r\nqueued body")); err != nil {
		t.Fatalf("queue inbound message: %v", err)
	}

	items, err := repo.ListByMailboxID(ctx, target.ID)
	if err != nil {
		t.Fatalf("list queued messages before worker: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no stored message before worker, got %d", len(items))
	}

	if err := jobs.RunInboundSpoolJob(ctx, service); err != nil {
		t.Fatalf("run inbound spool job: %v", err)
	}

	items, err = repo.ListByMailboxID(ctx, target.ID)
	if err != nil {
		t.Fatalf("list queued messages after worker: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 stored message after worker, got %d", len(items))
	}
	if items[0].Subject != "Async queued" {
		t.Fatalf("expected Async queued subject, got %s", items[0].Subject)
	}

	spoolItems, err := spool.List(ctx)
	if err != nil {
		t.Fatalf("list spool items: %v", err)
	}
	if len(spoolItems) != 1 {
		t.Fatalf("expected 1 spool item, got %d", len(spoolItems))
	}
	if spoolItems[0].Status != ingest.SpoolStatusCompleted {
		t.Fatalf("expected completed spool item, got %s", spoolItems[0].Status)
	}
}

func TestRunInboundSpoolJobLeavesRetriableFailureQueued(t *testing.T) {
	ctx := context.Background()
	mailboxRepo := mailbox.NewMemoryRepository()
	target, err := mailboxRepo.Create(ctx, mailbox.Mailbox{
		UserID:    1,
		DomainID:  1,
		Domain:    "example.test",
		LocalPart: "retry",
		Address:   "retry@example.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create retry mailbox: %v", err)
	}

	repo := ingest.NewMemoryMessageRepository()
	storage, err := ingest.NewLocalFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("create local storage: %v", err)
	}
	spool := ingest.NewMemorySpoolRepository()
	service := ingest.NewDirectService(mailboxRepo, repo, storage)
	service.SetSpoolRepository(spool)

	if _, err := service.Deliver(ctx, ingest.InboundEnvelope{
		MailFrom:   "sender@example.com",
		Recipients: []string{target.Address},
	}, strings.NewReader("From: sender@example.com\r\nTo: retry@example.test\r\nSubject: Retry\r\n\r\nqueued body")); err != nil {
		t.Fatalf("queue inbound message: %v", err)
	}

	if err := mailboxRepo.DeleteByID(ctx, target.ID); err != nil {
		t.Fatalf("delete mailbox before spool processing: %v", err)
	}

	err = jobs.RunInboundSpoolJob(ctx, service)
	if err == nil {
		t.Fatal("expected spool processing error after target removal")
	}

	items, err := spool.List(ctx)
	if err != nil {
		t.Fatalf("list spool items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 spool item, got %d", len(items))
	}
	if items[0].Status != ingest.SpoolStatusPending {
		t.Fatalf("expected failed item to be requeued, got %s", items[0].Status)
	}
	if items[0].AttemptCount != 1 {
		t.Fatalf("expected first attempt count 1, got %d", items[0].AttemptCount)
	}
}

func mailboxFixture(t *testing.T, id uint64, address string) mailbox.Mailbox {
	t.Helper()

	return mailbox.Mailbox{
		ID:        id,
		UserID:    1,
		DomainID:  1,
		Domain:    "example.test",
		LocalPart: address[:len(address)-len("@example.test")],
		Address:   address,
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

type stubLegacyMailSyncClient struct {
	headersByMailbox  map[string][]ingest.LegacyMessageHeader
	messagesByMailbox map[string]map[string]ingest.LegacyMessage
}

func (s stubLegacyMailSyncClient) ListMailbox(_ context.Context, mailboxName string) ([]ingest.LegacyMessageHeader, error) {
	return s.headersByMailbox[mailboxName], nil
}

func (s stubLegacyMailSyncClient) GetMessage(_ context.Context, mailboxName string, messageID string) (ingest.LegacyMessage, error) {
	return s.messagesByMailbox[mailboxName][messageID], nil
}
