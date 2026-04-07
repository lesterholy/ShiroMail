package bootstrap

import (
	"context"

	"shiro-email/backend/internal/jobs"
	"shiro-email/backend/internal/modules/mailbox"
	"shiro-email/backend/internal/modules/message"
)

func runWorkerCycle(ctx context.Context, mailboxRepo mailbox.Repository, messageRepo message.Repository, syncService jobs.MailboxSyncer) error {
	if err := jobs.RunSyncMessagesJob(ctx, mailboxRepo, syncService); err != nil {
		return err
	}
	return jobs.RunCleanupExpiredJob(ctx, mailboxRepo, messageRepo)
}
