package jobs

import (
	"context"
	"time"

	"shiro-email/backend/internal/modules/ingest"
	"shiro-email/backend/internal/modules/mailbox"
	"shiro-email/backend/internal/modules/message"
	"shiro-email/backend/internal/modules/system"
)

func RunCleanupExpiredJob(ctx context.Context, mailboxRepo mailbox.Repository, messageRepo message.Repository, configRepo system.ConfigRepository, storage ingest.FileStorage) error {
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
	if err := mailboxRepo.MarkExpired(ctx, expiredIDs); err != nil {
		return err
	}

	if storage == nil {
		return nil
	}
	settings, err := system.LoadMailInboundPolicySettings(ctx, configRepo)
	if err != nil {
		return err
	}
	if settings.RetainRawDays <= 0 {
		return nil
	}
	return storage.DeleteFilesOlderThan(ctx, time.Now().AddDate(0, 0, -settings.RetainRawDays))
}
