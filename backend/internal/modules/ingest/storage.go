package ingest

import (
	"context"
	"time"
)

type FileStorage interface {
	StoreRaw(ctx context.Context, mailboxAddress string, sourceMessageID string, raw []byte) (string, error)
	StoreAttachment(ctx context.Context, mailboxAddress string, sourceMessageID string, attachment InboundAttachment, index int) (StoredAttachment, error)
	ReadFile(ctx context.Context, key string) ([]byte, error)
	DeleteFilesOlderThan(ctx context.Context, before time.Time) error
}
