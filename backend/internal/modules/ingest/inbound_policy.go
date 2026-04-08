package ingest

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"shiro-email/backend/internal/modules/mailbox"
)

type InboundPolicy struct {
	MaxAttachmentSizeBytes int64
	RejectExecutableFiles  bool
}

type InboundPolicyProvider func(context.Context, []mailbox.Mailbox) (InboundPolicy, error)

type RejectionCode string

const (
	RejectAttachmentTooLarge   RejectionCode = "attachment_too_large"
	RejectExecutableAttachment RejectionCode = "attachment_type_rejected"
)

type RejectionError struct {
	Code    RejectionCode
	Message string
}

func (e *RejectionError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return fmt.Sprintf("inbound message rejected: %s", e.Code)
}

func IsRejectionCode(err error, code RejectionCode) bool {
	var rejectionErr *RejectionError
	return errors.As(err, &rejectionErr) && rejectionErr.Code == code
}

func isExecutableAttachment(attachment InboundAttachment) bool {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(attachment.FileName)))
	switch ext {
	case ".exe", ".msi", ".bat", ".cmd", ".scr", ".com", ".ps1", ".jar", ".dll", ".vbs", ".js":
		return true
	default:
		return false
	}
}
