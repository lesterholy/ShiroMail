package tests

import (
	"context"
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
