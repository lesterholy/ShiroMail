function titleCaseWords(value: string) {
  return value
    .split(/[\s_-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

type SMTPStatusDiagnosticFallback = {
  title: string;
  description: string;
  retryable: boolean;
};

export function describeSMTPRejectedReason(code: string) {
  const normalized = code.trim().toLowerCase();
  switch (normalized) {
    case "attachment_too_large":
      return {
        title: "Attachment Too Large",
        description:
          "The message was rejected because at least one attachment exceeded the active inbound size limit.",
        retryable: false,
      };
    case "executable_attachment_blocked":
      return {
        title: "Executable Attachment Blocked",
        description:
          "The inbound policy blocked an executable or script-like attachment before the message entered storage.",
        retryable: false,
      };
    case "mailbox_not_found":
      return {
        title: "Mailbox Not Found",
        description:
          "The RCPT target did not resolve to an active mailbox, so the SMTP session rejected delivery.",
        retryable: false,
      };
    case "invalid_recipient":
      return {
        title: "Invalid Recipient",
        description:
          "The recipient address format or routing target was invalid before the message could be accepted.",
        retryable: false,
      };
    default:
      return {
        title: titleCaseWords(normalized || "unknown_reject_reason"),
        description:
          "This reject counter is reported directly from the SMTP listener. Check recent spool or audit activity for more context.",
        retryable: false,
      };
  }
}

export function describeInboundSpoolFailure(message: string): SMTPStatusDiagnosticFallback {
  const normalized = message.trim().toLowerCase();
  if (!normalized) {
    return {
      title: "Unknown Failure",
      description:
        "The worker marked this spool item as failed without a detailed error string.",
      retryable: false,
    };
  }
  if (normalized.includes("mailbox not found")) {
    return {
      title: "Mailbox Not Found",
      description:
        "The message reached spool, but the worker could not match one or more target mailboxes during persistence.",
      retryable: false,
    };
  }
  if (normalized.includes("temporary parse failure")) {
    return {
      title: "Temporary Parse Failure",
      description:
        "The worker failed while parsing MIME content or message structure. This is often retryable after transient input or runtime issues clear.",
      retryable: true,
    };
  }
  if (normalized.includes("starttls")) {
    return {
      title: "Upstream STARTTLS Mismatch",
      description:
        "A delivery or relay path expected STARTTLS capability that was not advertised by the upstream server.",
      retryable: false,
    };
  }
  if (normalized.includes("auth")) {
    return {
      title: "Upstream Authentication Failure",
      description:
        "The worker or related delivery path hit an SMTP authentication issue. Verify credentials and AUTH capability.",
      retryable: false,
    };
  }
  return {
    title: message,
    description:
      "This failure message is reported directly from worker processing. Use the exact text to correlate logs and retry decisions.",
    retryable: false,
  };
}
