package system

import (
	"context"
	"errors"
	"testing"
)

type stubMailDeliveryTester struct {
	lastTo string
	err    error
}

func (s *stubMailDeliveryTester) SendTestMail(_ context.Context, to string) error {
	s.lastTo = to
	return s.err
}

func TestSendMailDeliveryTestUsesConfiguredFromAddressByDefault(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	tester := &stubMailDeliveryTester{}

	_, _ = configRepo.Upsert(context.Background(), ConfigKeyMailDelivery, map[string]any{
		"enabled":            true,
		"host":               "smtp.example.com",
		"port":               587,
		"username":           "sender@example.com",
		"password":           "app-password",
		"fromAddress":        "sender@example.com",
		"fromName":           "Shiro Email",
		"transportMode":      "starttls",
		"insecureSkipVerify": false,
	}, 1)

	service := NewService(configRepo, jobRepo, auditRepo, tester)

	result, err := service.SendMailDeliveryTest(context.Background(), 99, "")
	if err != nil {
		t.Fatalf("send test mail: %v", err)
	}
	if result.Recipient != "sender@example.com" {
		t.Fatalf("expected fallback recipient, got %q", result.Recipient)
	}
	if tester.lastTo != "sender@example.com" {
		t.Fatalf("expected tester recipient sender@example.com, got %q", tester.lastTo)
	}

	audits, err := auditRepo.List(context.Background())
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(audits) == 0 || audits[0].Action != "admin.mail_delivery.test" {
		t.Fatalf("expected audit log for mail delivery test, got %+v", audits)
	}
}

func TestSendMailDeliveryTestPreservesMailDeliveryError(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	tester := &stubMailDeliveryTester{
		err: wrapMailDeliveryError("auth", errors.New("server does not advertise AUTH")),
	}

	_, _ = configRepo.Upsert(context.Background(), ConfigKeyMailDelivery, map[string]any{
		"enabled":            true,
		"host":               "smtp.example.com",
		"port":               465,
		"username":           "sender@example.com",
		"password":           "secret",
		"fromAddress":        "sender@example.com",
		"fromName":           "Shiro Email",
		"transportMode":      "smtps",
		"insecureSkipVerify": false,
	}, 1)

	service := NewService(configRepo, jobRepo, auditRepo, tester)
	_, err := service.SendMailDeliveryTest(context.Background(), 99, "ops@example.com")
	if err == nil {
		t.Fatal("expected mail delivery test to fail")
	}

	var mailErr *MailDeliveryError
	if !errors.As(err, &mailErr) {
		t.Fatalf("expected MailDeliveryError, got %T %v", err, err)
	}
	if mailErr.Stage != "auth" {
		t.Fatalf("expected auth stage, got %+v", mailErr)
	}

	audits, auditErr := auditRepo.List(context.Background())
	if auditErr != nil {
		t.Fatalf("list audit logs: %v", auditErr)
	}
	if len(audits) == 0 || audits[0].Action != "admin.mail_delivery.test_failed" {
		t.Fatalf("expected failed mail delivery audit log, got %+v", audits)
	}
	if audits[0].Detail["code"] != "auth_unavailable" {
		t.Fatalf("expected auth_unavailable audit code, got %+v", audits[0].Detail)
	}
}

func TestListSettingsSectionsIncludesDynamicOAuthProviders(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	service := NewService(configRepo, jobRepo, auditRepo)

	_, _ = configRepo.Upsert(context.Background(), "auth.oauth.providers.discord", map[string]any{
		"displayName": "Discord",
		"enabled":     true,
	}, 1)

	sections, err := service.ListSettingsSections(context.Background())
	if err != nil {
		t.Fatalf("list settings sections: %v", err)
	}

	var oauthSection *SettingsSection
	for index := range sections {
		if sections[index].Key == "oauth" {
			oauthSection = &sections[index]
			break
		}
	}
	if oauthSection == nil {
		t.Fatal("expected oauth section to exist")
	}

	foundDisplay := false
	foundDiscord := false
	for _, item := range oauthSection.Items {
		if item.Key == ConfigKeyAuthOAuthDisplay {
			foundDisplay = true
		}
		if item.Key == "auth.oauth.providers.discord" {
			foundDiscord = true
		}
	}

	if !foundDisplay {
		t.Fatal("expected oauth section to include display config")
	}
	if !foundDiscord {
		t.Fatal("expected oauth section to include dynamic provider config")
	}
}

func TestListSettingsSectionsDoesNotRestoreDeletedDefaultOAuthProviders(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	service := NewService(configRepo, jobRepo, auditRepo)

	sections, err := service.ListSettingsSections(context.Background())
	if err != nil {
		t.Fatalf("list settings sections: %v", err)
	}

	var oauthSection *SettingsSection
	for index := range sections {
		if sections[index].Key == "oauth" {
			oauthSection = &sections[index]
			break
		}
	}
	if oauthSection == nil {
		t.Fatal("expected oauth section to exist")
	}

	if len(oauthSection.Items) != 1 {
		t.Fatalf("expected only oauth display config by default, got %d items", len(oauthSection.Items))
	}
	if oauthSection.Items[0].Key != ConfigKeyAuthOAuthDisplay {
		t.Fatalf("expected oauth display config only, got %q", oauthSection.Items[0].Key)
	}
}

func TestLoadAuthRuntimeSettingsDoesNotInjectDefaultOAuthProvidersFromEmptyRepo(t *testing.T) {
	configRepo := NewMemoryConfigRepository()

	settings, err := LoadAuthRuntimeSettings(context.Background(), configRepo)
	if err != nil {
		t.Fatalf("load auth runtime settings: %v", err)
	}

	if len(settings.OAuth) != 0 {
		t.Fatalf("expected no oauth providers from empty repo, got %+v", settings.OAuth)
	}
}

func TestLoadAPILimitsSettingsReturnsDefaultsFromEmptyRepo(t *testing.T) {
	configRepo := NewMemoryConfigRepository()

	settings, err := LoadAPILimitsSettings(context.Background(), configRepo)
	if err != nil {
		t.Fatalf("load api limits settings: %v", err)
	}

	if !settings.Enabled || settings.IdentityMode != "bearer_or_ip" {
		t.Fatalf("unexpected api limits defaults: %+v", settings)
	}
	if settings.AnonymousRPM != 120 || settings.AuthenticatedRPM != 600 || settings.MailboxWriteRPM != 1200 {
		t.Fatalf("unexpected api limits rates: %+v", settings)
	}
	if settings.ForgotPasswordRPM != 10 || settings.EmailVerificationResendRPM != 10 || settings.OAuthStartRPM != 20 {
		t.Fatalf("unexpected detailed api limits defaults: %+v", settings)
	}
}

func TestServiceAPILimitsSettingsLoadsStoredConfig(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	service := NewService(configRepo, jobRepo, auditRepo)

	_, _ = configRepo.Upsert(context.Background(), ConfigKeyAPILimits, map[string]any{
		"enabled":          true,
		"identityMode":     "ip",
		"anonymousRPM":     80,
		"authenticatedRPM": 400,
		"authRPM":          12,
		"loginRPM":         8,
		"registerRPM":      7,
		"refreshRPM":       25,
		"mailboxWriteRPM":  900,
		"strictIpEnabled":  true,
		"strictIpRPM":      1000,
	}, 1)

	settings, err := service.APILimitsSettings(context.Background())
	if err != nil {
		t.Fatalf("service api limits settings: %v", err)
	}
	if settings.IdentityMode != "ip" || settings.AnonymousRPM != 80 || !settings.StrictIPEnabled {
		t.Fatalf("unexpected api limit settings: %+v", settings)
	}
}

func TestListInboundSpoolReturnsConfiguredItems(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	service := NewService(configRepo, jobRepo, auditRepo, InboundSpoolListFunc(func(context.Context) ([]InboundSpoolRecord, error) {
		return []InboundSpoolRecord{{
			ID:               1,
			MailFrom:         "sender@example.com",
			Recipients:       []string{"queued@example.test"},
			TargetMailboxIDs: []uint64{7},
			Status:           "pending",
		}}, nil
	}))
	result, err := service.ListInboundSpool(context.Background(), InboundSpoolListOptions{})
	if err != nil {
		t.Fatalf("list inbound spool: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 inbound spool item, got %d", len(result.Items))
	}
	if result.Items[0].MailFrom != "sender@example.com" {
		t.Fatalf("expected sender@example.com, got %q", result.Items[0].MailFrom)
	}
	if result.Items[0].Status != "pending" {
		t.Fatalf("expected pending status, got %q", result.Items[0].Status)
	}
	if result.Total != 1 || result.Summary.Pending != 1 {
		t.Fatalf("unexpected inbound spool summary: %+v", result)
	}
	if len(result.FailureReasons) != 0 {
		t.Fatalf("expected no failure reasons, got %+v", result.FailureReasons)
	}
}

func TestRetryInboundSpoolUsesConfiguredHandlerAndAudits(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	service := NewService(configRepo, jobRepo, auditRepo, InboundSpoolRetryFunc(func(context.Context, uint64) (InboundSpoolRecord, error) {
		return InboundSpoolRecord{
			ID:           8,
			MailFrom:     "sender@example.com",
			Recipients:   []string{"queued@example.test"},
			Status:       "pending",
			AttemptCount: 0,
		}, nil
	}))

	item, err := service.RetryInboundSpool(context.Background(), 99, 8)
	if err != nil {
		t.Fatalf("retry inbound spool: %v", err)
	}
	if item.ID != 8 || item.Status != "pending" {
		t.Fatalf("unexpected retried spool item: %+v", item)
	}

	audits, err := auditRepo.List(context.Background())
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(audits) == 0 || audits[0].Action != "admin.inbound_spool.retry" {
		t.Fatalf("expected inbound spool retry audit log, got %+v", audits)
	}
}

func TestSMTPMetricsReturnsConfiguredSnapshot(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	service := NewService(configRepo, jobRepo, auditRepo, SMTPMetricsSnapshotFunc(func(context.Context) (SMTPMetricsSnapshot, error) {
		return SMTPMetricsSnapshot{
			SessionsStarted:    3,
			RecipientsAccepted: 5,
			BytesReceived:      2048,
			Accepted:           map[string]int64{"spool": 2},
			Rejected:           map[string]int64{"attachment_too_large": 1},
			SpoolProcessed:     map[string]int64{"completed": 2},
		}, nil
	}))

	item, err := service.SMTPMetrics(context.Background())
	if err != nil {
		t.Fatalf("load smtp metrics: %v", err)
	}
	if item.SessionsStarted != 3 || item.Accepted["spool"] != 2 || item.Rejected["attachment_too_large"] != 1 {
		t.Fatalf("unexpected smtp metrics snapshot: %+v", item)
	}
	if len(item.RejectedDetails) != 1 || item.RejectedDetails[0].Diagnostic.Title != "Attachment Too Large" {
		t.Fatalf("expected decorated rejected details, got %+v", item.RejectedDetails)
	}
}

func TestListInboundSpoolFiltersAndPaginates(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	service := NewService(configRepo, jobRepo, auditRepo, InboundSpoolListFunc(func(context.Context) ([]InboundSpoolRecord, error) {
		return []InboundSpoolRecord{
			{ID: 3, Status: "failed"},
			{ID: 2, Status: "pending"},
			{ID: 1, Status: "failed"},
		}, nil
	}))

	result, err := service.ListInboundSpool(context.Background(), InboundSpoolListOptions{
		Status:   "failed",
		Page:     2,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("list inbound spool with filters: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected 2 filtered items, got %d", result.Total)
	}
	if result.Page != 2 || result.PageSize != 1 {
		t.Fatalf("unexpected pagination metadata: %+v", result)
	}
	if len(result.Items) != 1 || result.Items[0].ID != 1 {
		t.Fatalf("unexpected paginated items: %+v", result.Items)
	}
	if result.Summary.Failed != 2 || result.Summary.Pending != 1 {
		t.Fatalf("unexpected summary counts: %+v", result.Summary)
	}
}

func TestListInboundSpoolAggregatesFailureReasons(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	service := NewService(configRepo, jobRepo, auditRepo, InboundSpoolListFunc(func(context.Context) ([]InboundSpoolRecord, error) {
		return []InboundSpoolRecord{
			{ID: 4, Status: "failed", ErrorMessage: "mailbox not found"},
			{ID: 3, Status: "failed", ErrorMessage: "temporary parse failure"},
			{ID: 2, Status: "failed", ErrorMessage: "mailbox not found"},
			{ID: 1, Status: "pending", ErrorMessage: "ignored while pending"},
		}, nil
	}))

	result, err := service.ListInboundSpool(context.Background(), InboundSpoolListOptions{})
	if err != nil {
		t.Fatalf("list inbound spool with failure reasons: %v", err)
	}
	if len(result.FailureReasons) != 2 {
		t.Fatalf("expected 2 failure reason buckets, got %+v", result.FailureReasons)
	}
	if result.FailureReasons[0].Message != "mailbox not found" || result.FailureReasons[0].Count != 2 {
		t.Fatalf("unexpected top failure reason: %+v", result.FailureReasons[0])
	}
	if result.FailureReasons[0].Diagnostic == nil || result.FailureReasons[0].Diagnostic.Title != "Mailbox Not Found" {
		t.Fatalf("expected decorated failure reason diagnostic, got %+v", result.FailureReasons[0])
	}
	if result.FailureReasons[1].Message != "temporary parse failure" || result.FailureReasons[1].Count != 1 {
		t.Fatalf("unexpected second failure reason: %+v", result.FailureReasons[1])
	}
	if result.Items[0].Diagnostic == nil && result.Items[1].Diagnostic == nil {
		t.Fatalf("expected spool items to include diagnostics, got %+v", result.Items)
	}
}

func TestListJobsDecoratesFailedInboundSpoolDiagnostics(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	service := NewService(configRepo, jobRepo, auditRepo)

	if _, err := jobRepo.Create(context.Background(), "inbound_spool", "failed", "temporary parse failure"); err != nil {
		t.Fatalf("create job: %v", err)
	}

	items, err := service.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one job, got %+v", items)
	}
	if items[0].Diagnostic == nil || items[0].Diagnostic.Title != "Temporary Parse Failure" || !items[0].Diagnostic.Retryable {
		t.Fatalf("expected decorated job diagnostic, got %+v", items[0])
	}
}

func TestListInboundSpoolFiltersByRetryableFailureMode(t *testing.T) {
	configRepo := NewMemoryConfigRepository()
	jobRepo := NewMemoryJobRepository()
	auditRepo := NewMemoryAuditRepository()
	service := NewService(configRepo, jobRepo, auditRepo, InboundSpoolListFunc(func(context.Context) ([]InboundSpoolRecord, error) {
		return []InboundSpoolRecord{
			{ID: 2, Status: "failed", ErrorMessage: "temporary parse failure"},
			{ID: 1, Status: "failed", ErrorMessage: "mailbox not found"},
		}, nil
	}))

	result, err := service.ListInboundSpool(context.Background(), InboundSpoolListOptions{
		Status:      "failed",
		FailureMode: "retryable",
	})
	if err != nil {
		t.Fatalf("list inbound spool with retryable filter: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ID != 2 {
		t.Fatalf("expected only retryable failure item, got %+v", result.Items)
	}
	if len(result.FailureReasons) != 1 || result.FailureReasons[0].Message != "temporary parse failure" {
		t.Fatalf("expected only retryable failure bucket, got %+v", result.FailureReasons)
	}
}
