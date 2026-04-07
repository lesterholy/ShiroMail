package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"shiro-email/backend/internal/modules/mailbox"
)

type InboundStore interface {
	StoreInbound(ctx context.Context, mailboxID uint64, item StoredInboundMessage) error
}

type DeliveryCallback func(mailboxUserID uint64, mailboxID uint64, mailboxAddress string, subject string)

type DirectService struct {
	mailboxes  mailbox.Repository
	store      InboundStore
	storage    FileStorage
	onDelivery DeliveryCallback
}

func NewDirectService(mailboxes mailbox.Repository, store InboundStore, storage FileStorage) *DirectService {
	return &DirectService{
		mailboxes: mailboxes,
		store:     store,
		storage:   storage,
	}
}

func (s *DirectService) SetDeliveryCallback(cb DeliveryCallback) {
	s.onDelivery = cb
}

func (s *DirectService) ResolveRecipient(ctx context.Context, address string) (mailbox.Mailbox, error) {
	return s.mailboxes.FindActiveByAddress(ctx, strings.ToLower(strings.TrimSpace(address)))
}

func (s *DirectService) Deliver(ctx context.Context, env InboundEnvelope, source io.Reader) (StoredInboundMessage, error) {
	targets, err := s.resolveTargets(ctx, env.Recipients)
	if err != nil {
		return StoredInboundMessage{}, err
	}
	return s.deliverToTargets(ctx, env, source, targets)
}

func (s *DirectService) DeliverResolved(ctx context.Context, env InboundEnvelope, source io.Reader, targets []mailbox.Mailbox) (StoredInboundMessage, error) {
	if len(targets) == 0 {
		return StoredInboundMessage{}, mailbox.ErrMailboxNotFound
	}
	return s.deliverToTargets(ctx, env, source, dedupeTargets(targets))
}

func (s *DirectService) resolveTargets(ctx context.Context, recipients []string) ([]mailbox.Mailbox, error) {
	targets := make([]mailbox.Mailbox, 0, len(recipients))
	for _, recipient := range recipients {
		target, err := s.ResolveRecipient(ctx, recipient)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return dedupeTargets(targets), nil
}

func dedupeTargets(targets []mailbox.Mailbox) []mailbox.Mailbox {
	if len(targets) <= 1 {
		return targets
	}

	seen := make(map[uint64]struct{}, len(targets))
	filtered := make([]mailbox.Mailbox, 0, len(targets))
	for _, target := range targets {
		if _, ok := seen[target.ID]; ok {
			continue
		}
		seen[target.ID] = struct{}{}
		filtered = append(filtered, target)
	}
	return filtered
}

func (s *DirectService) deliverToTargets(ctx context.Context, env InboundEnvelope, source io.Reader, targets []mailbox.Mailbox) (StoredInboundMessage, error) {
	if len(env.Recipients) == 0 {
		return StoredInboundMessage{}, fmt.Errorf("at least one recipient is required")
	}
	if len(targets) == 0 {
		return StoredInboundMessage{}, mailbox.ErrMailboxNotFound
	}

	parsed, err := ParseInboundMessage(env, source)
	if err != nil {
		return StoredInboundMessage{}, err
	}

	sourceMessageID := buildSourceMessageID(parsed.RawBytes)
	baseReceivedAt := parsed.ReceivedAt
	if baseReceivedAt.IsZero() {
		baseReceivedAt = time.Now().UTC()
	}

	deliveredTo := targets[0]
	lastItem := StoredInboundMessage{}
	for _, target := range targets {
		rawStorageKey, err := s.storage.StoreRaw(ctx, target.Address, sourceMessageID, parsed.RawBytes)
		if err != nil {
			return StoredInboundMessage{}, err
		}

		attachments := make([]StoredAttachment, 0, len(parsed.Attachments))
		for index, attachment := range parsed.Attachments {
			stored, err := s.storage.StoreAttachment(ctx, target.Address, sourceMessageID, attachment, index)
			if err != nil {
				return StoredInboundMessage{}, err
			}
			attachments = append(attachments, stored)
		}

		item := StoredInboundMessage{
			SourceKind:      "smtp",
			SourceMessageID: sourceMessageID,
			MailboxAddress:  target.Address,
			FromAddr:        firstNonEmpty(parsed.FromAddr, env.MailFrom),
			ToAddr:          firstNonEmpty(parsed.ToAddr, strings.Join(env.Recipients, ","), target.Address),
			Subject:         parsed.Subject,
			TextPreview:     buildPreview(parsed.TextBody),
			HTMLPreview:     parsed.HTMLBody,
			TextBody:        parsed.TextBody,
			HTMLBody:        parsed.HTMLBody,
			Headers:         parsed.Headers,
			RawStorageKey:   rawStorageKey,
			HasAttachments:  len(attachments) > 0,
			SizeBytes:       int64(len(parsed.RawBytes)),
			ReceivedAt:      baseReceivedAt,
			Attachments:     attachments,
		}

		if err := s.store.StoreInbound(ctx, target.ID, item); err != nil {
			return StoredInboundMessage{}, err
		}
		if s.onDelivery != nil {
			s.onDelivery(target.UserID, target.ID, target.Address, item.Subject)
		}
		lastItem = item
		deliveredTo = target
	}

	lastItem.MailboxAddress = deliveredTo.Address
	return lastItem, nil
}

func buildSourceMessageID(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:16])
}

func buildPreview(body string) string {
	trimmed := strings.TrimSpace(body)
	if utf8.RuneCountInString(trimmed) <= 160 {
		return trimmed
	}

	count := 0
	for index := range trimmed {
		if count == 160 {
			return trimmed[:index]
		}
		count += 1
	}
	return trimmed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
