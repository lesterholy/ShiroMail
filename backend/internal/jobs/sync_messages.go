package jobs

import (
	"context"

	"shiro-email/backend/internal/modules/mailbox"
)

type MailboxSyncer interface {
	SyncMailbox(ctx context.Context, target mailbox.Mailbox) error
}

func RunSyncMessagesJob(ctx context.Context, repo mailbox.Repository, syncService MailboxSyncer) error {
	if syncService == nil {
		return nil
	}

	items, err := repo.ListActive(ctx)
	if err != nil {
		return err
	}

	for _, item := range items {
		if err := syncService.SyncMailbox(ctx, item); err != nil {
			return err
		}
	}

	return nil
}
