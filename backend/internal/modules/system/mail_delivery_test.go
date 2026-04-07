package system

import (
	"bytes"
	"net/mail"
	"strings"
	"testing"

	"github.com/jhillyerd/enmime/v2"
)

func TestBuildMailDeliveryCodeMessageVerificationHTML(t *testing.T) {
	message := BuildMailDeliveryCodeMessage(
		"Shiro Email verification code",
		"Your verification code is:",
		"123456",
		"http://localhost:5173/auth/verify-email?ticket=ticket-1&code=123456",
		"Verify email",
	)

	if !strings.Contains(message.HTMLBody, "完成账户邮箱验证") {
		t.Fatalf("expected verification headline in HTML body")
	}
	if !strings.Contains(message.HTMLBody, "123456") {
		t.Fatalf("expected code in HTML body")
	}
	if !strings.Contains(message.HTMLBody, "邮箱验证") {
		t.Fatalf("expected verification badge in HTML body")
	}
	if !strings.Contains(message.HTMLBody, "Verify email") || !strings.Contains(message.HTMLBody, "/auth/verify-email?ticket=ticket-1&amp;code=123456") {
		t.Fatalf("expected verification action link in HTML body")
	}
	if len(message.TextLines) == 0 || !strings.Contains(message.TextLines[0], "123456") {
		t.Fatalf("expected plain text fallback with code, got %+v", message.TextLines)
	}
}

func TestBuildMailDeliveryCodeMessagePasswordResetHTML(t *testing.T) {
	message := BuildMailDeliveryCodeMessage(
		"Shiro Email password reset code",
		"Your password reset code is:",
		"654321",
		"http://localhost:5173/auth/reset-password?ticket=ticket-2&code=654321",
		"Reset password",
	)

	if !strings.Contains(message.HTMLBody, "重置你的账户密码") {
		t.Fatalf("expected password reset headline in HTML body")
	}
	if !strings.Contains(message.HTMLBody, "重置密码") {
		t.Fatalf("expected password reset badge in HTML body")
	}
	if !strings.Contains(message.HTMLBody, "#dc2626") {
		t.Fatalf("expected password reset accent color in HTML body")
	}
	if !strings.Contains(message.HTMLBody, "Reset password") || !strings.Contains(message.HTMLBody, "/auth/reset-password?ticket=ticket-2&amp;code=654321") {
		t.Fatalf("expected password reset action link in HTML body")
	}
	if len(message.TextLines) < 3 || !strings.Contains(message.TextLines[2], "检查账户安全") {
		t.Fatalf("expected security warning in text fallback, got %+v", message.TextLines)
	}
}

func TestBuildMailDeliveryMIMEBodyCreatesMultipartAlternative(t *testing.T) {
	settings := MailDeliveryConfig{
		FromAddress: "sender@example.com",
		FromName:    "Shiro Email",
	}
	body, err := buildMailDeliveryMIMEBody(settings, "user@example.com", "Subject", MailDeliveryMessage{
		TextLines: []string{"Plain body"},
		HTMLBody:  "<html><body><strong>HTML body</strong></body></html>",
	})
	if err != nil {
		t.Fatalf("build MIME body: %v", err)
	}

	raw := string(body)
	if !strings.Contains(raw, "multipart/alternative") {
		t.Fatalf("expected multipart content type, got %s", raw)
	}
	if !strings.Contains(raw, "Content-Transfer-Encoding: quoted-printable") {
		t.Fatalf("expected quoted-printable body, got %s", raw)
	}
	if !strings.Contains(raw, "Content-Type: text/plain; charset=UTF-8") {
		t.Fatalf("expected text/plain part")
	}
	if !strings.Contains(raw, "Content-Type: text/html; charset=UTF-8") {
		t.Fatalf("expected text/html part")
	}
	if !strings.Contains(raw, "HTML body") {
		t.Fatalf("expected HTML body content")
	}
}

func TestBuildMailDeliveryMIMEBodyRoundTripsAsMultipartEmail(t *testing.T) {
	settings := MailDeliveryConfig{
		FromAddress: "sender@example.com",
		FromName:    "Shiro Email",
	}
	body, err := buildMailDeliveryMIMEBody(settings, "user@example.com", "验证码测试", MailDeliveryMessage{
		TextLines: []string{
			"Your verification code is: 014455",
			"验证码 15 分钟内有效。",
		},
		HTMLBody: "<!DOCTYPE html><html><body><strong>014455</strong><p>验证码 15 分钟内有效。</p></body></html>",
	})
	if err != nil {
		t.Fatalf("build MIME body: %v", err)
	}

	headerMessage, err := mail.ReadMessage(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("parse top-level message: %v", err)
	}
	if got := headerMessage.Header.Get("Content-Type"); !strings.Contains(got, "multipart/alternative") {
		t.Fatalf("expected multipart content type header, got %q", got)
	}
	if got := headerMessage.Header.Get("Subject"); !strings.Contains(got, "=?utf-8?q?") {
		t.Fatalf("expected encoded subject header, got %q", got)
	}

	envelope, err := enmime.ReadEnvelope(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("parse MIME envelope: %v", err)
	}
	if !strings.Contains(envelope.Text, "014455") {
		t.Fatalf("expected decoded plain text body, got %q", envelope.Text)
	}
	if !strings.Contains(envelope.HTML, "<strong>014455</strong>") {
		t.Fatalf("expected decoded html body, got %q", envelope.HTML)
	}
}

func TestBuildMailDeliveryTestMessageHTML(t *testing.T) {
	message := BuildMailDeliveryTestMessage("2026-04-06T10:00:00Z")

	if !strings.Contains(message.HTMLBody, "SMTP 发信测试成功") {
		t.Fatalf("expected SMTP test headline in HTML body")
	}
	if !strings.Contains(message.HTMLBody, "已连接") {
		t.Fatalf("expected connected status in HTML body")
	}
	if !strings.Contains(message.HTMLBody, "SMTP 测试") {
		t.Fatalf("expected SMTP test badge in HTML body")
	}
	if len(message.TextLines) < 2 || !strings.Contains(message.TextLines[1], "2026-04-06T10:00:00Z") {
		t.Fatalf("expected sent time in text fallback, got %+v", message.TextLines)
	}
}
