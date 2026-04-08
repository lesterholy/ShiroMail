package smtp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/textproto"
	"strings"
	"time"

	"shiro-email/backend/internal/middleware"
	"shiro-email/backend/internal/modules/ingest"
	"shiro-email/backend/internal/modules/mailbox"
)

type Deliverer interface {
	ResolveRecipient(ctx context.Context, address string) (mailbox.Mailbox, error)
	Deliver(ctx context.Context, env ingest.InboundEnvelope, source io.Reader) (ingest.StoredInboundMessage, error)
}

type ResolvedDeliverer interface {
	DeliverResolved(ctx context.Context, env ingest.InboundEnvelope, source io.Reader, targets []mailbox.Mailbox) (ingest.StoredInboundMessage, error)
}

type sessionState int

const (
	stateGreet sessionState = iota
	stateReady
	stateMail
)

type session struct {
	ctx        context.Context
	cfg        Config
	deliver    Deliverer
	conn       net.Conn
	text       *textproto.Conn
	state      sessionState
	mailFrom   string
	recipients []string
	targets    []mailbox.Mailbox
}

func newSession(ctx context.Context, cfg Config, deliver Deliverer, conn net.Conn) *session {
	return &session{
		ctx:     ctx,
		cfg:     cfg,
		deliver: deliver,
		conn:    conn,
		text:    textproto.NewConn(conn),
		state:   stateGreet,
	}
}

func (s *session) run() {
	defer s.conn.Close()
	middleware.RecordSMTPSessionStarted()
	s.writeLine(fmt.Sprintf("220 %s Shiro SMTP ready", s.cfg.Hostname))

	for {
		line, err := s.readLine()
		if err != nil {
			return
		}

		cmd, arg := parseCommand(line)
		switch cmd {
		case "QUIT":
			s.writeLine("221 Bye")
			return
		case "RSET":
			s.resetEnvelope()
			s.state = stateReady
			s.writeLine("250 Session reset")
		case "HELO":
			if strings.TrimSpace(arg) == "" {
				s.writeLine("501 Domain/address argument required for HELO")
				continue
			}
			s.state = stateReady
			s.writeLine("250 Hello")
		case "EHLO":
			if strings.TrimSpace(arg) == "" {
				s.writeLine("501 Domain/address argument required for EHLO")
				continue
			}
			s.state = stateReady
			s.writeLine("250-Hello")
			s.writeLine("250 SIZE " + fmt.Sprintf("%d", s.cfg.MaxMessageBytes))
		case "MAIL":
			if s.state != stateReady {
				s.writeLine("503 Bad sequence of commands")
				continue
			}
			address, ok := parsePathArg(arg, "FROM:")
			if !ok {
				s.writeLine("501 Was expecting MAIL arg syntax of FROM:<address>")
				continue
			}
			s.resetEnvelope()
			s.mailFrom = address
			s.state = stateMail
			s.writeLine("250 Sender OK")
		case "RCPT":
			if s.state != stateMail {
				s.writeLine("503 Bad sequence of commands")
				continue
			}
			address, ok := parsePathArg(arg, "TO:")
			if !ok {
				s.writeLine("501 Was expecting RCPT arg syntax of TO:<address>")
				continue
			}
			target, err := s.deliver.ResolveRecipient(s.ctx, address)
			if err != nil {
				if !isMailboxLookupFailure(err) {
					slog.Warn(
						"smtp recipient resolution failed",
						"remote",
						s.conn.RemoteAddr().String(),
						"recipient",
						address,
						"normalized_recipient",
						strings.ToLower(strings.TrimSpace(address)),
						"mail_from",
						s.mailFrom,
						"envelope_recipient_count",
						len(s.recipients)+1,
						"stage",
						"rcpt_to",
						"error",
						err,
					)
					s.writeLine("550 Delivery failed")
					middleware.RecordSMTPDeliveryRejected("recipient_resolution_failed")
					s.resetEnvelope()
					s.state = stateReady
					continue
				}
				slog.Info(
					"smtp recipient rejected",
					"remote",
					s.conn.RemoteAddr().String(),
					"recipient",
					address,
					"normalized_recipient",
					strings.ToLower(strings.TrimSpace(address)),
					"mail_from",
					s.mailFrom,
					"envelope_recipient_count",
					len(s.recipients)+1,
					"stage",
					"rcpt_to",
					"reason",
					"recipient_not_found_or_inactive_or_expired",
					"hint",
					"recipient lookup only accepts active, unexpired local mailboxes",
				)
				s.writeLine("550 Relay not permitted")
				middleware.RecordSMTPDeliveryRejected("recipient_not_found_or_inactive_or_expired")
				continue
			}
			s.recipients = append(s.recipients, strings.ToLower(address))
			s.targets = append(s.targets, target)
			middleware.RecordSMTPRecipientAccepted()
			slog.Info("smtp recipient accepted", "remote", s.conn.RemoteAddr().String(), "recipient", address, "mailboxID", target.ID)
			s.writeLine("250 Recipient OK")
		case "DATA":
			if s.state != stateMail || len(s.recipients) == 0 {
				s.writeLine("503 Bad sequence of commands")
				continue
			}
			s.handleData()
			s.state = stateReady
		default:
			s.writeLine("502 Command not implemented")
		}
	}
}

func (s *session) handleData() {
	s.writeLine("354 Start mail input; end with <CRLF>.<CRLF>")
	body, err := s.readData()
	if err != nil {
		s.writeLine("451 Failed to read message")
		s.resetEnvelope()
		return
	}
	if len(body) > s.cfg.MaxMessageBytes {
		middleware.RecordSMTPDeliveryRejected("message_too_large")
		s.writeLine("552 Max message size exceeded")
		s.resetEnvelope()
		return
	}
	middleware.RecordSMTPDeliveryBytes(len(body))
	env := ingest.InboundEnvelope{
		MailFrom:   s.mailFrom,
		Recipients: append([]string{}, s.recipients...),
	}

	var (
		storedItem ingest.StoredInboundMessage
		deliverErr error
	)
	if resolvedDeliverer, ok := s.deliver.(ResolvedDeliverer); ok && len(s.targets) != 0 {
		storedItem, deliverErr = resolvedDeliverer.DeliverResolved(s.ctx, env, bytes.NewReader(body), append([]mailbox.Mailbox{}, s.targets...))
	} else {
		storedItem, deliverErr = s.deliver.Deliver(s.ctx, env, bytes.NewReader(body))
	}
	if deliverErr != nil {
		if isMailboxLookupFailure(deliverErr) {
			slog.Info(
				"smtp delivery rejected",
				"remote",
				s.conn.RemoteAddr().String(),
				"mail_from",
				s.mailFrom,
				"recipients",
				strings.Join(s.recipients, ","),
				"recipient_count",
				len(s.recipients),
				"stage",
				"data",
				"reason",
				"recipient_not_found_or_inactive_or_expired",
				"hint",
				"recipient lookup only accepts active, unexpired local mailboxes",
				"error",
				deliverErr,
			)
			s.writeLine("550 Relay not permitted")
			middleware.RecordSMTPDeliveryRejected("recipient_not_found_or_inactive_or_expired")
		} else if ingest.IsRejectionCode(deliverErr, ingest.RejectAttachmentTooLarge) {
			slog.Info(
				"smtp delivery rejected by inbound policy",
				"remote",
				s.conn.RemoteAddr().String(),
				"mail_from",
				s.mailFrom,
				"recipients",
				strings.Join(s.recipients, ","),
				"recipient_count",
				len(s.recipients),
				"stage",
				"data",
				"reason",
				"attachment_too_large",
				"error",
				deliverErr,
			)
			s.writeLine("552 Attachment exceeds policy limit")
			middleware.RecordSMTPDeliveryRejected("attachment_too_large")
		} else if ingest.IsRejectionCode(deliverErr, ingest.RejectExecutableAttachment) {
			slog.Info(
				"smtp delivery rejected by inbound policy",
				"remote",
				s.conn.RemoteAddr().String(),
				"mail_from",
				s.mailFrom,
				"recipients",
				strings.Join(s.recipients, ","),
				"recipient_count",
				len(s.recipients),
				"stage",
				"data",
				"reason",
				"attachment_type_rejected",
				"error",
				deliverErr,
			)
			s.writeLine("550 Attachment type rejected")
			middleware.RecordSMTPDeliveryRejected("attachment_type_rejected")
		} else {
			slog.Error("smtp delivery failed", "remote", s.conn.RemoteAddr().String(), "recipients", strings.Join(s.recipients, ","), "error", deliverErr)
			s.writeLine("451 Failed to store message")
			middleware.RecordSMTPDeliveryRejected("store_failed")
		}
		s.resetEnvelope()
		return
	}
	if storedItem.SourceKind == "smtp-spool" {
		middleware.RecordSMTPDeliveryAccepted("spool")
		slog.Info(
			"smtp delivery queued",
			"remote",
			s.conn.RemoteAddr().String(),
			"recipients",
			strings.Join(s.recipients, ","),
			"bytes",
			len(body),
			"source_message_id",
			storedItem.SourceMessageID,
		)
	} else {
		middleware.RecordSMTPDeliveryAccepted("direct")
		slog.Info("smtp delivery accepted", "remote", s.conn.RemoteAddr().String(), "recipients", strings.Join(s.recipients, ","), "bytes", len(body))
	}
	s.writeLine("250 Mail accepted for delivery")
	s.resetEnvelope()
}

func (s *session) readLine() (string, error) {
	_ = s.conn.SetReadDeadline(time.Now().Add(s.cfg.Timeout))
	return s.text.ReadLine()
}

func (s *session) readData() ([]byte, error) {
	_ = s.conn.SetReadDeadline(time.Now().Add(s.cfg.Timeout))
	return s.text.ReadDotBytes()
}

func (s *session) writeLine(line string) {
	_ = s.conn.SetWriteDeadline(time.Now().Add(s.cfg.Timeout))
	_ = s.text.PrintfLine("%s", line)
}

func (s *session) resetEnvelope() {
	s.mailFrom = ""
	s.recipients = nil
	s.targets = nil
}

func parseCommand(line string) (string, string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", ""
	}
	if idx := strings.IndexByte(line, ' '); idx >= 0 {
		return strings.ToUpper(line[:idx]), strings.TrimSpace(line[idx+1:])
	}
	return strings.ToUpper(line), ""
}

func parsePathArg(arg string, prefix string) (string, bool) {
	if !strings.HasPrefix(strings.ToUpper(arg), prefix) {
		return "", false
	}
	value := strings.TrimSpace(arg[len(prefix):])
	value = strings.Trim(value, "<> ")
	if value == "" {
		return "", false
	}
	return value, true
}

func isMailboxLookupFailure(err error) bool {
	return errors.Is(err, mailbox.ErrMailboxNotFound)
}
