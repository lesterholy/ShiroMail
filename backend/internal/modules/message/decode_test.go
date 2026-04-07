package message

import (
	"testing"
	"time"

	"shiro-email/backend/internal/database"
)

func TestMapMessageSummaryDecodesMIMEHeaders(t *testing.T) {
	row := database.MessageRow{
		ID:          1,
		MailboxID:   2,
		FromAddr:    "=?UTF-8?B?5pyo5YG2?= <sender@example.com>",
		Subject:     "=?UTF-8?B?5Zu+54mH5rWL6K+V?=",
		TextPreview: "body",
		ReceivedAt:  time.Now(),
	}

	item := mapMessageSummary(row, 0)
	if item.FromAddr != "木偶 <sender@example.com>" {
		t.Fatalf("expected decoded from address, got %q", item.FromAddr)
	}
	if item.Subject != "图片测试" {
		t.Fatalf("expected decoded subject, got %q", item.Subject)
	}
}
