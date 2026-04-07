package ingest

import (
	"bytes"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/jhillyerd/enmime/v2"
	"shiro-email/backend/internal/shared/mimeheader"
)

type InboundEnvelope struct {
	MailFrom   string
	Recipients []string
}

type InboundAttachment struct {
	FileName    string
	ContentType string
	ContentID   string
	Content     []byte
	SizeBytes   int64
}

type InboundMessage struct {
	Envelope    InboundEnvelope
	Headers     map[string][]string
	Subject     string
	FromAddr    string
	ToAddr      string
	TextBody    string
	HTMLBody    string
	RawBytes    []byte
	ReceivedAt  time.Time
	Attachments []InboundAttachment
}

func ParseInboundMessage(env InboundEnvelope, source io.Reader) (InboundMessage, error) {
	rawBytes, err := io.ReadAll(source)
	if err != nil {
		return InboundMessage{}, err
	}

	headerMessage, err := mail.ReadMessage(bytes.NewReader(rawBytes))
	if err != nil {
		return InboundMessage{}, err
	}
	parsedEnvelope, err := enmime.ReadEnvelope(bytes.NewReader(rawBytes))
	if err != nil {
		return InboundMessage{}, err
	}

	receivedAt := time.Time{}
	if dateHeader := headerMessage.Header.Get("Date"); dateHeader != "" {
		if parsedDate, dateErr := mail.ParseDate(dateHeader); dateErr == nil {
			receivedAt = parsedDate.UTC()
		}
	}

	attachments := make([]InboundAttachment, 0, len(parsedEnvelope.Inlines)+len(parsedEnvelope.Attachments))
	for _, part := range append([]*enmime.Part{}, parsedEnvelope.Inlines...) {
		attachments = append(attachments, mapInboundAttachment(part))
	}
	for _, part := range parsedEnvelope.Attachments {
		attachments = append(attachments, mapInboundAttachment(part))
	}

	return InboundMessage{
		Envelope:    env,
		Headers:     cloneHeaders(headerMessage.Header),
		Subject:     mimeheader.Decode(headerMessage.Header.Get("Subject")),
		FromAddr:    mimeheader.Decode(headerMessage.Header.Get("From")),
		ToAddr:      mimeheader.Decode(headerMessage.Header.Get("To")),
		TextBody:    strings.TrimSpace(parsedEnvelope.Text),
		HTMLBody:    strings.TrimSpace(parsedEnvelope.HTML),
		RawBytes:    rawBytes,
		ReceivedAt:  receivedAt,
		Attachments: attachments,
	}, nil
}

func cloneHeaders(header mail.Header) map[string][]string {
	cloned := make(map[string][]string, len(header))
	for key, values := range header {
		cloned[key] = mimeheader.DecodeValues(values)
	}
	return cloned
}

func mapInboundAttachment(part *enmime.Part) InboundAttachment {
	content := make([]byte, len(part.Content))
	copy(content, part.Content)
	return InboundAttachment{
		FileName:    part.FileName,
		ContentType: part.ContentType,
		ContentID:   normalizeContentID(part.ContentID),
		Content:     content,
		SizeBytes:   int64(len(content)),
	}
}

func normalizeContentID(value string) string {
	return strings.Trim(strings.TrimSpace(value), "<>")
}
