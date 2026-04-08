# SMTP Hardening And Delivery Roadmap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade ShiroMail SMTP from a basic embedded listener into a policy-aware, observable, and extensible mail pipeline without breaking current inbound delivery.

**Architecture:** Keep the existing embedded SMTP server for now, but add a policy enforcement layer between MIME parsing and persistence, enrich SMTP reject semantics, and stage later work into async spool, retention cleanup, and outbound provider abstraction. The first implementation slice focuses on enforcing inbound attachment policy at runtime and surfacing precise SMTP responses/log reasons.

**Tech Stack:** Go 1.24, Gin, GORM, custom SMTP server, enmime MIME parser, MySQL-backed system config, existing backend integration tests.

---

## Phase Breakdown

### P0 — Immediate Hardening
- [ ] Load `mail.inbound_policy` into runtime code instead of leaving it as UI-only config.
- [ ] Enforce attachment size limits during inbound SMTP processing.
- [ ] Reject executable attachments when policy requires it.
- [ ] Return precise SMTP rejection codes/messages for policy failures.
- [ ] Enrich logs with structured rejection reasons for attachment policy failures.
- [ ] Add regression tests for direct ingest policy rejection and SMTP surface behavior.

### P1 — Reliability And Operability
- [ ] Introduce an inbound spool/job table so SMTP `DATA` only depends on raw persistence, not full parsing + DB writes.
- [ ] Add worker-driven asynchronous parsing and retry/dead-letter behavior.
- [ ] Implement raw/attachment retention cleanup using `retainRawDays`.
- [ ] Add SMTP metrics and aggregate reject/success counters.
- [ ] Add domain-level override hooks for inbound policy.

### P2 — Delivery Platform Enhancements
- [ ] Add outbound sender abstraction for STARTTLS / SMTPS / provider-aware delivery.
- [ ] Add SPF/DKIM/DMARC guidance and validation feedback loops.
- [ ] Add anti-abuse controls: IP/session rate limiting, RCPT quotas, greylist/RBL hooks.
- [ ] Split SMTP into an optional dedicated process when the single-process model becomes an operational bottleneck.

## Current Implementation Notes

- Runtime inbound policy enforcement is already live, including attachment size rejection and executable attachment rejection.
- Async inbound spool, worker retry flow, retention cleanup, and SMTP metrics/admin visibility are already landed.
- Outbound delivery now supports `plain`, `starttls`, and `smtps`, with strict capability failure on missing `STARTTLS` / `AUTH`.
- SMTP delivery test failures now expose structured diagnostics:
  - `stage`: delivery phase such as `connect`, `tls`, `auth`, `rcpt_to`
  - `code`: normalized reason such as `starttls_unavailable` or `recipient_rejected`
  - `hint`: operator-facing remediation guidance
  - `retryable`: whether retrying without config changes is likely to help
- Admin UI alignment should keep these diagnostics visible in both the settings page and recent audit activity.

## Task 1: Runtime Inbound Policy Enforcement

**Files:**
- Modify: `backend/internal/modules/system/mail_delivery.go`
- Modify: `backend/internal/bootstrap/app.go`
- Modify: `backend/internal/modules/ingest/direct_service.go`
- Modify: `backend/internal/modules/ingest/smtp/session.go`
- Test: `backend/internal/modules/ingest/direct_service_test.go`
- Test: `backend/internal/modules/ingest/smtp/server_test.go`

**Intent:** turn `mail.inbound_policy` into live runtime behavior for attachment governance.

- [ ] Add a loader for `MailInboundPolicyConfig` with normalized defaults.
- [ ] Introduce ingest-level typed rejection errors for policy-driven message rejection.
- [ ] Wire a policy provider into `DirectService` so runtime policy can be resolved without hard-coding system package dependencies into SMTP session handling.
- [ ] Validate parsed attachments before persistence:
  - reject attachments larger than configured limit
  - reject executable attachments when policy says so
- [ ] Map those typed errors to explicit SMTP responses (`552` for size, `550` for blocked type).
- [ ] Emit structured logs that distinguish mailbox lookup rejection from policy rejection.

## Task 2: Retention And File Lifecycle

**Files:**
- Modify: `backend/internal/modules/system/mail_delivery.go`
- Modify: `backend/internal/modules/ingest/file_storage.go`
- Modify: `backend/internal/bootstrap/worker.go`
- Modify: `backend/internal/jobs/cleanup_expired.go`
- Test: `backend/tests/message_integration_test.go`

**Intent:** make `retainRawDays` affect on-disk raw and attachment files, not just UI config.

- [ ] Expose retention config as a worker-consumable runtime value.
- [ ] Add file storage deletion primitives and safe path cleanup.
- [ ] Extend cleanup job to prune aged raw/attachment artifacts.
- [ ] Preserve active mailbox artifacts while deleting orphaned/expired ones.
- [ ] Add regression tests for file retention cleanup.

## Task 3: Async Inbound Spool

**Files:**
- Create: `backend/internal/modules/ingest/spool.go`
- Create: `backend/internal/modules/ingest/mysql_spool_repository.go`
- Modify: `backend/internal/modules/ingest/direct_service.go`
- Modify: `backend/internal/bootstrap/app.go`
- Modify: `backend/internal/bootstrap/worker.go`
- Test: `backend/tests/ingest_integration_test.go`

**Intent:** decouple SMTP accept latency from parse/store latency.

- [ ] Persist raw inbound payload into a spool queue during SMTP `DATA`.
- [ ] Move MIME parsing and DB persistence into worker execution.
- [ ] Add retry and terminal-failure states.
- [ ] Keep current websocket/webhook behavior after successful worker completion.
- [ ] Add regression tests for queue-backed inbound processing.

## Task 4: Outbound Sender Abstraction

**Files:**
- Modify: `backend/internal/modules/system/mail_delivery.go`
- Modify: `backend/internal/modules/auth/email_delivery.go`
- Create: `backend/internal/modules/system/outbound_sender.go`
- Test: `backend/internal/modules/system/mail_delivery_test.go`

**Intent:** move from `smtp.SendMail` straight calls to a configurable sender layer.

- [ ] Add explicit transport modes for plaintext SMTP, STARTTLS, and SMTPS.
- [ ] Add dial/auth timeout handling and reusable connection behavior.
- [ ] Keep current config schema backward-compatible.
- [ ] Add unit tests for MIME generation and transport selection.

## Execution Order
- [ ] Execute Task 1 first; it is self-contained, low-risk, and improves current production safety immediately.
- [ ] Only start Task 2 after Task 1 tests are green.
- [ ] Only start Task 3 after Task 1 and Task 2 are stable, because async spool changes the pipeline architecture.
- [ ] Keep Task 4 independent so it can land in parallel later if needed.
