package system

import "strings"

type SMTPStatusDiagnostic struct {
	Code        string `json:"code,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Retryable   bool   `json:"retryable"`
}

func DiagnoseSMTPRejectedReason(code string) SMTPStatusDiagnostic {
	normalized := strings.ToLower(strings.TrimSpace(code))
	switch normalized {
	case "attachment_too_large":
		return SMTPStatusDiagnostic{
			Code:        normalized,
			Title:       "Attachment Too Large",
			Description: "The message was rejected because at least one attachment exceeded the active inbound size limit.",
			Retryable:   false,
		}
	case "executable_attachment_blocked":
		return SMTPStatusDiagnostic{
			Code:        normalized,
			Title:       "Executable Attachment Blocked",
			Description: "The inbound policy blocked an executable or script-like attachment before the message entered storage.",
			Retryable:   false,
		}
	case "mailbox_not_found":
		return SMTPStatusDiagnostic{
			Code:        normalized,
			Title:       "Mailbox Not Found",
			Description: "The RCPT target did not resolve to an active mailbox, so the SMTP session rejected delivery.",
			Retryable:   false,
		}
	case "invalid_recipient":
		return SMTPStatusDiagnostic{
			Code:        normalized,
			Title:       "Invalid Recipient",
			Description: "The recipient address format or routing target was invalid before the message could be accepted.",
			Retryable:   false,
		}
	default:
		return SMTPStatusDiagnostic{
			Code:        normalized,
			Title:       titleCaseWords(normalized, "unknown_reject_reason"),
			Description: "This reject counter is reported directly from the SMTP listener. Check recent spool or audit activity for more context.",
			Retryable:   false,
		}
	}
}

func DiagnoseInboundSpoolFailure(message string) SMTPStatusDiagnostic {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return SMTPStatusDiagnostic{
			Title:       "Unknown Failure",
			Description: "The worker marked this spool item as failed without a detailed error string.",
			Retryable:   false,
		}
	}
	if strings.Contains(normalized, "mailbox not found") {
		return SMTPStatusDiagnostic{
			Code:        "mailbox_not_found",
			Title:       "Mailbox Not Found",
			Description: "The message reached spool, but the worker could not match one or more target mailboxes during persistence.",
			Retryable:   false,
		}
	}
	if strings.Contains(normalized, "temporary parse failure") {
		return SMTPStatusDiagnostic{
			Code:        "temporary_parse_failure",
			Title:       "Temporary Parse Failure",
			Description: "The worker failed while parsing MIME content or message structure. This is often retryable after transient input or runtime issues clear.",
			Retryable:   true,
		}
	}
	if strings.Contains(normalized, "starttls") {
		return SMTPStatusDiagnostic{
			Code:        "starttls_unavailable",
			Title:       "Upstream STARTTLS Mismatch",
			Description: "A delivery or relay path expected STARTTLS capability that was not advertised by the upstream server.",
			Retryable:   false,
		}
	}
	if strings.Contains(normalized, "auth") {
		return SMTPStatusDiagnostic{
			Code:        "auth_failed",
			Title:       "Upstream Authentication Failure",
			Description: "The worker or related delivery path hit an SMTP authentication issue. Verify credentials and AUTH capability.",
			Retryable:   false,
		}
	}
	return SMTPStatusDiagnostic{
		Title:       strings.TrimSpace(message),
		Description: "This failure message is reported directly from worker processing. Use the exact text to correlate logs and retry decisions.",
		Retryable:   false,
	}
}

func titleCaseWords(value string, fallback string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		normalized = fallback
	}
	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ' ' || r == '_' || r == '-'
	})
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
