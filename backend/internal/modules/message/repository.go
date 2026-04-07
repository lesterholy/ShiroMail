package message

import (
	"context"

	"shiro-email/backend/internal/modules/ingest"
)

type Repository interface {
	UpsertFromLegacySync(ctx context.Context, mailboxID uint64, mailboxName string, parsed ingest.ParsedMessage) error
	StoreInbound(ctx context.Context, mailboxID uint64, item ingest.StoredInboundMessage) error
	ListSummaryByMailboxID(ctx context.Context, mailboxID uint64) ([]Summary, error)
	SearchSummaryByMailboxID(ctx context.Context, mailboxID uint64, query string) ([]Summary, error)
	ListByMailboxID(ctx context.Context, mailboxID uint64) ([]Message, error)
	SearchByMailboxID(ctx context.Context, mailboxID uint64, query string) ([]Message, error)
	GetByMailboxAndID(ctx context.Context, mailboxID uint64, messageID uint64) (Message, error)
	SoftDeleteByMailboxIDs(ctx context.Context, mailboxIDs []uint64) error
	CountToday(ctx context.Context) int
}

type MemoryRepository struct {
	*ingest.MemoryMessageRepository
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{MemoryMessageRepository: ingest.NewMemoryMessageRepository()}
}

func (r *MemoryRepository) ListSummaryByMailboxID(ctx context.Context, mailboxID uint64) ([]Summary, error) {
	items, err := r.ListByMailboxID(ctx, mailboxID)
	if err != nil {
		return nil, err
	}
	return summarizeMessages(items), nil
}

func (r *MemoryRepository) SearchSummaryByMailboxID(ctx context.Context, mailboxID uint64, query string) ([]Summary, error) {
	items, err := r.SearchByMailboxID(ctx, mailboxID, query)
	if err != nil {
		return nil, err
	}
	return summarizeMessages(items), nil
}
