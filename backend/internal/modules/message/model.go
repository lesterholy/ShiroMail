package message

import (
	"time"

	"shiro-email/backend/internal/modules/ingest"
)

type Message = ingest.StoredMessage

type Attachment = ingest.StoredAttachment

type Summary struct {
	ID               uint64    `json:"id"`
	MailboxID        uint64    `json:"mailboxId"`
	LegacyMailboxKey string    `json:"legacyMailboxKey"`
	LegacyMessageKey string    `json:"legacyMessageKey"`
	SourceKind       string    `json:"sourceKind"`
	SourceMessageID  string    `json:"sourceMessageId"`
	MailboxAddress   string    `json:"mailboxAddress"`
	FromAddr         string    `json:"fromAddr"`
	ToAddr           string    `json:"toAddr"`
	Subject          string    `json:"subject"`
	TextPreview      string    `json:"textPreview"`
	HTMLPreview      string    `json:"htmlPreview"`
	HasAttachments   bool      `json:"hasAttachments"`
	AttachmentCount  int       `json:"attachmentCount"`
	SizeBytes        int64     `json:"sizeBytes"`
	IsRead           bool      `json:"isRead"`
	IsDeleted        bool      `json:"isDeleted"`
	ReceivedAt       time.Time `json:"receivedAt"`
}

type ParsedRawAttachment struct {
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	ContentID   string `json:"contentId"`
	SizeBytes   int64  `json:"sizeBytes"`
}

type ParsedRawMessage struct {
	MessageID       uint64                `json:"messageId"`
	MailboxID       uint64                `json:"mailboxId"`
	Subject         string                `json:"subject"`
	FromAddr        string                `json:"fromAddr"`
	ToAddr          string                `json:"toAddr"`
	ReceivedAt      time.Time             `json:"receivedAt"`
	TextBody        string                `json:"textBody"`
	HTMLBody        string                `json:"htmlBody"`
	Headers         map[string][]string   `json:"headers"`
	AttachmentCount int                   `json:"attachmentCount"`
	Attachments     []ParsedRawAttachment `json:"attachments"`
	RawSizeBytes    int64                 `json:"rawSizeBytes"`
}

type ReceiveRawMessageRequest struct {
	MailFrom string `json:"mailFrom"`
	Raw      string `json:"raw" binding:"required"`
}
