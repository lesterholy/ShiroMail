package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	"shiro-email/backend/internal/modules/ingest"
	"shiro-email/backend/internal/modules/mailbox"
	"shiro-email/backend/internal/modules/message"
	"shiro-email/backend/internal/modules/system"
)

func TestResolveInboundPolicyForTargetsUsesDomainOverrideForSingleDomain(t *testing.T) {
	policy := resolveInboundPolicyForTargets(system.MailInboundPolicyConfig{
		MaxAttachmentSizeMB:   15,
		RejectExecutableFiles: true,
		DomainOverrides: map[string]system.MailInboundPolicyDomainConfig{
			"example.test": {
				Enabled:               true,
				MaxAttachmentSizeMB:   5,
				RejectExecutableFiles: false,
			},
		},
	}, []mailbox.Mailbox{{
		Domain: "example.test",
	}})

	if policy.MaxAttachmentSizeBytes != 5*1024*1024 {
		t.Fatalf("expected domain override size 5 MB, got %d", policy.MaxAttachmentSizeBytes)
	}
	if policy.RejectExecutableFiles {
		t.Fatal("expected single-domain override to relax executable rejection")
	}
}

func TestResolveInboundPolicyForTargetsMergesMultipleDomainsRestrictively(t *testing.T) {
	policy := resolveInboundPolicyForTargets(system.MailInboundPolicyConfig{
		MaxAttachmentSizeMB:   15,
		RejectExecutableFiles: false,
		DomainOverrides: map[string]system.MailInboundPolicyDomainConfig{
			"alpha.test": {
				Enabled:               true,
				MaxAttachmentSizeMB:   5,
				RejectExecutableFiles: false,
			},
			"beta.test": {
				Enabled:               true,
				MaxAttachmentSizeMB:   8,
				RejectExecutableFiles: true,
			},
		},
	}, []mailbox.Mailbox{
		{Domain: "alpha.test"},
		{Domain: "beta.test"},
	})

	if policy.MaxAttachmentSizeBytes != 5*1024*1024 {
		t.Fatalf("expected stricter 5 MB policy, got %d", policy.MaxAttachmentSizeBytes)
	}
	if !policy.RejectExecutableFiles {
		t.Fatal("expected merged policy to reject executable attachments")
	}
}

func TestRunWorkerCycleSkipsLegacySyncAndStillCleansExpiredMailboxes(t *testing.T) {
	mailboxRepo := mailbox.NewMemoryRepository()
	messageRepo := message.NewMemoryRepository()
	jobRepo := system.NewMemoryJobRepository()

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

	if err := runWorkerCycle(context.Background(), mailboxRepo, messageRepo, nil, nil, nil, nil, jobRepo); err != nil {
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

func TestRunWorkerCycleRecordsInboundSpoolFailureJob(t *testing.T) {
	mailboxRepo := mailbox.NewMemoryRepository()
	messageRepo := message.NewMemoryRepository()
	jobRepo := system.NewMemoryJobRepository()
	storage, err := ingest.NewLocalFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("create local storage: %v", err)
	}

	target, err := mailboxRepo.Create(context.Background(), mailbox.Mailbox{
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
		t.Fatalf("create mailbox: %v", err)
	}

	service := ingest.NewDirectService(mailboxRepo, messageRepo, storage)
	service.SetSpoolRepository(ingest.NewMemorySpoolRepository())
	if _, err := service.Deliver(context.Background(), ingest.InboundEnvelope{
		MailFrom:   "sender@example.com",
		Recipients: []string{target.Address},
	}, strings.NewReader("From: sender@example.com\r\nTo: retry@example.test\r\nSubject: Retry\r\n\r\nbody")); err != nil {
		t.Fatalf("queue inbound message: %v", err)
	}
	if err := mailboxRepo.DeleteByID(context.Background(), target.ID); err != nil {
		t.Fatalf("delete mailbox before worker: %v", err)
	}

	if err := runWorkerCycle(context.Background(), mailboxRepo, messageRepo, nil, nil, service, nil, jobRepo); err == nil {
		t.Fatal("expected worker cycle to surface inbound spool failure")
	}

	items, err := jobRepo.List(context.Background())
	if err != nil {
		t.Fatalf("list job records: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected failed job record")
	}
	if items[0].JobType != "inbound_spool" {
		t.Fatalf("expected inbound_spool job type, got %s", items[0].JobType)
	}
	if items[0].Status != "failed" {
		t.Fatalf("expected failed job status, got %s", items[0].Status)
	}
}
