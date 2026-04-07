package jobs

import (
	"context"
	"time"

	"shiro-email/backend/internal/modules/mailbox"
	"shiro-email/backend/internal/modules/message"
)

func RunCleanupExpiredJob(ctx context.Context, mailboxRepo mailbox.Repository, messageRepo message.Repository) error {
	expiredIDs, err := mailboxRepo.ListExpiredIDs(ctx, time.Now())
	if err != nil {
		return err
	}
	if len(expiredIDs) == 0 {
		return nil
	}
	if err := messageRepo.SoftDeleteByMailboxIDs(ctx, expiredIDs); err != nil {
		return err
	}
	return mailboxRepo.MarkExpired(ctx, expiredIDs)
}
