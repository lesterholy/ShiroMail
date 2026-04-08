package tests

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"shiro-email/backend/internal/bootstrap"
	"shiro-email/backend/internal/modules/ingest"
)

func TestAdminCanListInboundSpoolItems(t *testing.T) {
	server, state := bootstrap.NewTestApp()

	spool := ingest.NewMemorySpoolRepository()
	state.DirectIngest.SetSpoolRepository(spool)
	if _, err := spool.Enqueue(context.Background(), ingest.SpoolItem{
		MailFrom:         "sender@example.com",
		Recipients:       []string{"queued@example.test"},
		TargetMailboxIDs: []uint64{1},
		RawMessage:       []byte("raw"),
	}); err != nil {
		t.Fatalf("enqueue spool item: %v", err)
	}

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"admin","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected admin login success, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")

	rr := performJSON(server, http.MethodGet, "/api/v1/admin/jobs/inbound-spool", "", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"mailFrom":"sender@example.com"`) {
		t.Fatalf("expected inbound spool sender in response, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"status":"pending"`) {
		t.Fatalf("expected pending inbound spool status, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"summary":{"total":1,"pending":1`) {
		t.Fatalf("expected inbound spool summary in response, got %s", rr.Body.String())
	}
}

func TestAdminCanRetryInboundSpoolItems(t *testing.T) {
	server, state := bootstrap.NewTestApp()

	spool := ingest.NewMemorySpoolRepository()
	state.DirectIngest.SetSpoolRepository(spool)
	if _, err := spool.Enqueue(context.Background(), ingest.SpoolItem{
		MailFrom:         "sender@example.com",
		Recipients:       []string{"queued@example.test"},
		TargetMailboxIDs: []uint64{1},
		RawMessage:       []byte("raw"),
	}); err != nil {
		t.Fatalf("enqueue spool item: %v", err)
	}

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"admin","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected admin login success, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")

	rr := performJSON(server, http.MethodPost, "/api/v1/admin/jobs/inbound-spool/1/retry", "{}", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"status":"pending"`) {
		t.Fatalf("expected pending retried status, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"attemptCount":0`) {
		t.Fatalf("expected attempt count reset in response, got %s", rr.Body.String())
	}
}

func TestAdminCanFilterInboundSpoolItems(t *testing.T) {
	server, state := bootstrap.NewTestApp()

	spool := ingest.NewMemorySpoolRepository()
	state.DirectIngest.SetSpoolRepository(spool)
	if _, err := spool.Enqueue(context.Background(), ingest.SpoolItem{
		MailFrom:         "pending@example.com",
		Recipients:       []string{"queued@example.test"},
		TargetMailboxIDs: []uint64{1},
		RawMessage:       []byte("raw"),
	}); err != nil {
		t.Fatalf("enqueue pending spool item: %v", err)
	}
	if _, err := spool.Enqueue(context.Background(), ingest.SpoolItem{
		MailFrom:         "processing@example.com",
		Recipients:       []string{"queued@example.test"},
		TargetMailboxIDs: []uint64{1},
		RawMessage:       []byte("raw"),
	}); err != nil {
		t.Fatalf("enqueue processing spool item: %v", err)
	}
	if _, err := spool.ClaimNext(context.Background()); err != nil {
		t.Fatalf("claim spool item: %v", err)
	}

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"admin","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected admin login success, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")

	rr := performJSON(server, http.MethodGet, "/api/v1/admin/jobs/inbound-spool?status=processing&page=1&pageSize=1", "", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"total":1`) {
		t.Fatalf("expected filtered total 1, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"processing":1`) {
		t.Fatalf("expected processing summary count, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"mailFrom":"pending@example.com"`) {
		t.Fatalf("expected processing spool item in filtered response, got %s", rr.Body.String())
	}
}

func TestAdminInboundSpoolIncludesFailureReasonSummary(t *testing.T) {
	server, state := bootstrap.NewTestApp()

	spool := ingest.NewMemorySpoolRepository()
	state.DirectIngest.SetSpoolRepository(spool)
	if _, err := spool.Enqueue(context.Background(), ingest.SpoolItem{
		MailFrom:         "failed1@example.com",
		Recipients:       []string{"queued@example.test"},
		TargetMailboxIDs: []uint64{1},
		RawMessage:       []byte("raw"),
		MaxAttempts:      1,
	}); err != nil {
		t.Fatalf("enqueue first failed spool item: %v", err)
	}
	if _, err := spool.Enqueue(context.Background(), ingest.SpoolItem{
		MailFrom:         "failed2@example.com",
		Recipients:       []string{"queued@example.test"},
		TargetMailboxIDs: []uint64{1},
		RawMessage:       []byte("raw"),
		MaxAttempts:      1,
	}); err != nil {
		t.Fatalf("enqueue second failed spool item: %v", err)
	}
	if _, err := spool.ClaimNext(context.Background()); err != nil {
		t.Fatalf("claim first spool item: %v", err)
	}
	if err := spool.MarkFailed(context.Background(), 1, "mailbox not found"); err != nil {
		t.Fatalf("mark first spool item failed attempt 1: %v", err)
	}
	if _, err := spool.ClaimNext(context.Background()); err != nil {
		t.Fatalf("claim second spool item: %v", err)
	}
	if err := spool.MarkFailed(context.Background(), 2, "mailbox not found"); err != nil {
		t.Fatalf("mark second spool item failed attempt 1: %v", err)
	}

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"admin","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected admin login success, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")

	rr := performJSON(server, http.MethodGet, "/api/v1/admin/jobs/inbound-spool", "", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"failureReasons":[{"message":"mailbox not found","count":2,`) {
		t.Fatalf("expected aggregated failure reasons in response, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"diagnostic":{"code":"mailbox_not_found","title":"Mailbox Not Found"`) {
		t.Fatalf("expected failure diagnostics in response, got %s", rr.Body.String())
	}
}

func TestAdminCanReadSMTPMetricsSnapshot(t *testing.T) {
	server, _ := bootstrap.NewTestApp()

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"admin","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected admin login success, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")

	rr := performJSON(server, http.MethodGet, "/api/v1/admin/jobs/smtp-metrics", "", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"sessionsStarted"`) {
		t.Fatalf("expected smtp metrics snapshot payload, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"accepted"`) {
		t.Fatalf("expected smtp accepted counters in response, got %s", rr.Body.String())
	}
}

func TestAdminCanFilterInboundSpoolByRetryableFailures(t *testing.T) {
	server, state := bootstrap.NewTestApp()

	spool := ingest.NewMemorySpoolRepository()
	state.DirectIngest.SetSpoolRepository(spool)
	if _, err := spool.Enqueue(context.Background(), ingest.SpoolItem{
		MailFrom:         "retryable@example.com",
		Recipients:       []string{"queued@example.test"},
		TargetMailboxIDs: []uint64{1},
		RawMessage:       []byte("raw"),
		MaxAttempts:      1,
	}); err != nil {
		t.Fatalf("enqueue retryable spool item: %v", err)
	}
	if _, err := spool.Enqueue(context.Background(), ingest.SpoolItem{
		MailFrom:         "nonretryable@example.com",
		Recipients:       []string{"queued@example.test"},
		TargetMailboxIDs: []uint64{1},
		RawMessage:       []byte("raw"),
		MaxAttempts:      1,
	}); err != nil {
		t.Fatalf("enqueue non-retryable spool item: %v", err)
	}
	if _, err := spool.ClaimNext(context.Background()); err != nil {
		t.Fatalf("claim retryable spool item: %v", err)
	}
	if err := spool.MarkFailed(context.Background(), 1, "temporary parse failure"); err != nil {
		t.Fatalf("mark retryable spool item failed: %v", err)
	}
	if _, err := spool.ClaimNext(context.Background()); err != nil {
		t.Fatalf("claim non-retryable spool item: %v", err)
	}
	if err := spool.MarkFailed(context.Background(), 2, "mailbox not found"); err != nil {
		t.Fatalf("mark non-retryable spool item failed: %v", err)
	}

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"admin","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected admin login success, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")

	rr := performJSON(server, http.MethodGet, "/api/v1/admin/jobs/inbound-spool?status=failed&failureMode=retryable", "", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"mailFrom":"retryable@example.com"`) {
		t.Fatalf("expected retryable spool item in response, got %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"mailFrom":"nonretryable@example.com"`) {
		t.Fatalf("did not expect non-retryable spool item in response, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"retryable":true`) {
		t.Fatalf("expected retryable diagnostic in response, got %s", rr.Body.String())
	}
}

func TestAdminMailDeliveryTestReturnsStructuredDiagnostic(t *testing.T) {
	server, state := bootstrap.NewTestApp()

	_, _ = state.ConfigRepo.Upsert(context.Background(), "mail.delivery", map[string]any{
		"enabled":            true,
		"host":               "127.0.0.1",
		"port":               1,
		"username":           "",
		"password":           "",
		"fromAddress":        "sender@example.com",
		"fromName":           "Shiro Email",
		"transportMode":      "plain",
		"insecureSkipVerify": false,
	}, 1)

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"admin","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected admin login success, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")

	rr := performJSON(server, http.MethodPost, "/api/v1/admin/configs/mail.delivery/test", `{"to":"ops@example.com"}`, token)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"code":"connect_failed"`) {
		t.Fatalf("expected structured diagnostic code in response, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"hint":"`) {
		t.Fatalf("expected structured diagnostic hint in response, got %s", rr.Body.String())
	}
}
