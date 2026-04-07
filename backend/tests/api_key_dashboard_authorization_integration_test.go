package tests

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"shiro-email/backend/internal/bootstrap"
	"shiro-email/backend/internal/modules/mailbox"
	"shiro-email/backend/internal/modules/portal"
)

func seedOwnedDashboardMailbox(t *testing.T, state *bootstrap.AppState, server http.Handler, username string, userToken string, domainName string, localPart string) (uint64, uint64) {
	t.Helper()

	rr := performJSON(server, http.MethodPost, "/api/v1/domains", `{"domain":"`+domainName+`","status":"active"}`, userToken)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected domain create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}

	domainID := parseUint64(t, extractJSONScalarField(rr.Body.String(), "id"))
	user, err := state.AuthRepo.FindUserByLogin(context.Background(), username)
	if err != nil {
		t.Fatalf("expected auth user lookup success, got %v", err)
	}

	item, err := state.MailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    user.ID,
		DomainID:  domainID,
		Domain:    domainName,
		LocalPart: localPart,
		Address:   localPart + "@" + domainName,
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected mailbox seed success, got %v", err)
	}

	return domainID, item.ID
}

func TestAPIKeyCanAccessDashboardWithReadScopes(t *testing.T) {
	server, state := newTestServerWithState(t)
	userToken := registerAndLogin(t, server, "api-key-dashboard-user")
	_, mailboxID := seedOwnedDashboardMailbox(t, state, server, "api-key-dashboard-user", userToken, "dashboard-owned.test", "dash")

	apiKey := createAPIKeyPreview(t, state, "api-key-dashboard-user", portal.CreateAPIKeyInput{
		Name:   "dashboard-reader",
		Scopes: []string{"domains.read", "mailboxes.read"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode:         "private_only",
			AllowOwnedPrivateDomains: true,
		},
	})

	rr := performJSON(server, http.MethodGet, "/api/v1/dashboard", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected dashboard with read scopes to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"domain":"dashboard-owned.test"`) {
		t.Fatalf("expected owned domain in dashboard response: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"id":`+strconv.FormatUint(mailboxID, 10)) {
		t.Fatalf("expected owned mailbox in dashboard response: %s", rr.Body.String())
	}
}

func TestAPIKeyCannotAccessDashboardWithoutDomainReadScope(t *testing.T) {
	server, state := newTestServerWithState(t)
	userToken := registerAndLogin(t, server, "api-key-dashboard-missing-scope")
	seedOwnedDashboardMailbox(t, state, server, "api-key-dashboard-missing-scope", userToken, "dashboard-scope.test", "dash")

	apiKey := createAPIKeyPreview(t, state, "api-key-dashboard-missing-scope", portal.CreateAPIKeyInput{
		Name:   "mailbox-only-reader",
		Scopes: []string{"mailboxes.read"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode:         "private_only",
			AllowOwnedPrivateDomains: true,
		},
	})

	rr := performJSON(server, http.MethodGet, "/api/v1/dashboard", "", apiKey)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected dashboard without domains.read to return 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAPIKeyDashboardHonorsDomainBinding(t *testing.T) {
	server, state := newTestServerWithState(t)
	userToken := registerAndLogin(t, server, "api-key-dashboard-bound")
	firstDomainID, _ := seedOwnedDashboardMailbox(t, state, server, "api-key-dashboard-bound", userToken, "dashboard-bound-one.test", "one")
	_, _ = seedOwnedDashboardMailbox(t, state, server, "api-key-dashboard-bound", userToken, "dashboard-bound-two.test", "two")

	boundNodeID := firstDomainID
	apiKey := createAPIKeyPreview(t, state, "api-key-dashboard-bound", portal.CreateAPIKeyInput{
		Name:   "bound-dashboard-reader",
		Scopes: []string{"domains.read", "mailboxes.read"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode:         "private_only",
			AllowOwnedPrivateDomains: true,
		},
		DomainBindings: []portal.APIKeyDomainBinding{
			{
				NodeID:      &boundNodeID,
				AccessLevel: "read",
			},
		},
	})

	rr := performJSON(server, http.MethodGet, "/api/v1/dashboard", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected bound dashboard request to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"domain":"dashboard-bound-one.test"`) {
		t.Fatalf("expected bound domain in dashboard response: %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"domain":"dashboard-bound-two.test"`) {
		t.Fatalf("expected unbound domain to be filtered out: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"address":"one@dashboard-bound-one.test"`) {
		t.Fatalf("expected bound mailbox in dashboard response: %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"address":"two@dashboard-bound-two.test"`) {
		t.Fatalf("expected unbound mailbox to be filtered out: %s", rr.Body.String())
	}
}
