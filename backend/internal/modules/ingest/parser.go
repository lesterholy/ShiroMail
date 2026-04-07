package ingest

import (
	"strings"
	"time"
)

type ParsedMessage struct {
	LegacyMailboxKey string
	LegacyMessageKey string
	FromAddr         string
	ToAddr           string
	Subject          string
	TextPreview      string
	HTMLPreview      string
	SizeBytes        int64
	IsRead           bool
	ReceivedAt       time.Time
	Header           map[string][]string
	Attachments      []ParsedAttachment
}

type ParsedAttachment struct {
	FileName    string
	ContentType string
	StorageKey  string
	SizeBytes   int64
}

func ParseLegacyRawMessage(raw LegacyMessage) ParsedMessage {
	receivedAt := raw.Date
	if receivedAt.IsZero() && raw.PosixMillis > 0 {
		receivedAt = time.UnixMilli(raw.PosixMillis).UTC()
	}

	textPreview := ""
	htmlPreview := ""
	if raw.Body != nil {
		textPreview = strings.TrimSpace(raw.Body.Text)
		htmlPreview = strings.TrimSpace(raw.Body.HTML)
	}

	attachments := make([]ParsedAttachment, 0, len(raw.Attachments))
	for _, item := range raw.Attachments {
		storageKey := item.DownloadLink
		if storageKey == "" {
			storageKey = item.ViewLink
		}
		if storageKey == "" {
			storageKey = item.MD5
		}
		attachments = append(attachments, ParsedAttachment{
			FileName:    item.FileName,
			ContentType: item.ContentType,
			StorageKey:  storageKey,
		})
	}

	return ParsedMessage{
		LegacyMailboxKey: raw.Mailbox,
		LegacyMessageKey: raw.ID,
		FromAddr:         raw.From,
		ToAddr:           strings.Join(raw.To, ","),
		Subject:          raw.Subject,
		TextPreview:      textPreview,
		HTMLPreview:      htmlPreview,
		SizeBytes:        raw.Size,
		IsRead:           raw.Seen,
		ReceivedAt:       receivedAt,
		Header:           raw.Header,
		Attachments:      attachments,
	}
}
