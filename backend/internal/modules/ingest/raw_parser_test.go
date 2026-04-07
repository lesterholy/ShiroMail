package ingest

import (
	"strings"
	"testing"
)

func TestParseInboundMessageExtractsBodiesAndAttachments(t *testing.T) {
	raw := strings.NewReader("From: sender@example.com\r\nTo: alpha@example.test\r\nSubject: Welcome\r\nDate: Thu, 02 Apr 2026 09:30:00 +0800\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=abc\r\n\r\n--abc\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nhello from smtp\r\n--abc\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<p>hello from smtp</p>\r\n--abc\r\nContent-Type: text/plain\r\nContent-Disposition: attachment; filename=\"note.txt\"\r\n\r\nattachment body\r\n--abc--\r\n")

	parsed, err := ParseInboundMessage(InboundEnvelope{
		MailFrom:   "sender@example.com",
		Recipients: []string{"alpha@example.test"},
	}, raw)
	if err != nil {
		t.Fatalf("parse inbound message: %v", err)
	}
	if parsed.Subject != "Welcome" {
		t.Fatalf("expected subject Welcome, got %s", parsed.Subject)
	}
	if parsed.FromAddr != "sender@example.com" {
		t.Fatalf("expected sender@example.com, got %s", parsed.FromAddr)
	}
	if parsed.ToAddr != "alpha@example.test" {
		t.Fatalf("expected alpha@example.test, got %s", parsed.ToAddr)
	}
	if parsed.TextBody != "hello from smtp" {
		t.Fatalf("expected text body, got %q", parsed.TextBody)
	}
	if parsed.HTMLBody != "<p>hello from smtp</p>" {
		t.Fatalf("expected html body, got %q", parsed.HTMLBody)
	}
	if len(parsed.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(parsed.Attachments))
	}
	if parsed.Attachments[0].FileName != "note.txt" {
		t.Fatalf("expected note.txt attachment, got %s", parsed.Attachments[0].FileName)
	}
	if string(parsed.Attachments[0].Content) != "attachment body" {
		t.Fatalf("unexpected attachment content %q", string(parsed.Attachments[0].Content))
	}
	if parsed.Headers["Subject"][0] != "Welcome" {
		t.Fatalf("expected Subject header, got %+v", parsed.Headers["Subject"])
	}
	if len(parsed.RawBytes) == 0 {
		t.Fatal("expected raw bytes to be preserved")
	}
	if parsed.ReceivedAt.IsZero() {
		t.Fatal("expected parsed received time from Date header")
	}
}

func TestParseInboundMessageExtractsInlineContentID(t *testing.T) {
	raw := strings.NewReader("From: sender@example.com\r\nTo: alpha@example.test\r\nSubject: Inline image\r\nMIME-Version: 1.0\r\nContent-Type: multipart/related; boundary=abc\r\n\r\n--abc\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<img src=\"cid:logo@test\">\r\n--abc\r\nContent-Type: image/png\r\nContent-Transfer-Encoding: base64\r\nContent-ID: <logo@test>\r\nContent-Disposition: inline; filename=\"logo.png\"\r\n\r\naGVsbG8=\r\n--abc--\r\n")

	parsed, err := ParseInboundMessage(InboundEnvelope{
		MailFrom:   "sender@example.com",
		Recipients: []string{"alpha@example.test"},
	}, raw)
	if err != nil {
		t.Fatalf("parse inbound message: %v", err)
	}
	if len(parsed.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(parsed.Attachments))
	}
	if parsed.Attachments[0].ContentID != "logo@test" {
		t.Fatalf("expected normalized content id logo@test, got %q", parsed.Attachments[0].ContentID)
	}
}

func TestParseInboundMessageDecodesMIMEHeaders(t *testing.T) {
	raw := strings.NewReader("From: =?UTF-8?B?5pyo5YG2?= <sender@example.com>\r\nTo: alpha@example.test\r\nSubject: =?UTF-8?B?5Zu+54mH5rWL6K+V?=\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nhello\r\n")

	parsed, err := ParseInboundMessage(InboundEnvelope{
		MailFrom:   "sender@example.com",
		Recipients: []string{"alpha@example.test"},
	}, raw)
	if err != nil {
		t.Fatalf("parse inbound message: %v", err)
	}
	if parsed.FromAddr != "木偶 <sender@example.com>" {
		t.Fatalf("expected decoded from address, got %q", parsed.FromAddr)
	}
	if parsed.Subject != "图片测试" {
		t.Fatalf("expected decoded subject, got %q", parsed.Subject)
	}
	if parsed.Headers["From"][0] != "木偶 <sender@example.com>" {
		t.Fatalf("expected decoded From header, got %+v", parsed.Headers["From"])
	}
	if parsed.Headers["Subject"][0] != "图片测试" {
		t.Fatalf("expected decoded Subject header, got %+v", parsed.Headers["Subject"])
	}
}
