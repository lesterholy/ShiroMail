package tests

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"shiro-email/backend/internal/bootstrap"
	"shiro-email/backend/internal/modules/portal"
)

func newTestServerWithState(t *testing.T) (http.Handler, *bootstrap.AppState) {
	t.Helper()
	server, state := bootstrap.NewTestApp()
	return server, state
}

func createAPIKeyPreview(t *testing.T, state *bootstrap.AppState, username string, input portal.CreateAPIKeyInput) string {
	t.Helper()

	user, err := state.AuthRepo.FindUserByLogin(context.Background(), username)
	if err != nil {
		t.Fatalf("expected seeded auth user, got %v", err)
	}

	service := portal.NewService(state.PortalRepo, state.AuthRepo)
	item, err := service.CreateAPIKey(context.Background(), user.ID, input)
	if err != nil {
		t.Fatalf("expected api key create success, got %v", err)
	}
	if item.PlainSecret == "" {
		t.Fatalf("expected api key preview, got %+v", item)
	}
	return item.PlainSecret
}

func parseUint64(t *testing.T, value string) uint64 {
	t.Helper()

	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		t.Fatalf("expected valid uint64 value %q, got %v", value, err)
	}
	return parsed
}

func TestAPIKeyCanAccessDomainListWithDomainsReadScope(t *testing.T) {
	server, state := newTestServerWithState(t)
	userToken := registerAndLogin(t, server, "api-key-domain-reader")

	rr := performJSON(server, http.MethodPost, "/api/v1/domains", `{"domain":"owned-api-key.test","status":"active"}`, userToken)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected owned domain create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}

	apiKey := createAPIKeyPreview(t, state, "api-key-domain-reader", portal.CreateAPIKeyInput{
		Name:   "domain-reader",
		Scopes: []string{"domains.read"},
	})

	rr = performJSON(server, http.MethodGet, "/api/v1/domains", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected api key domain list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"domain":"owned-api-key.test"`) {
		t.Fatalf("expected owned domain in api key domain list: %s", rr.Body.String())
	}
}

func TestAPIKeyCannotCreateMailboxWithoutMailboxesWriteScope(t *testing.T) {
	server, state := newTestServerWithState(t)
	_ = registerAndLogin(t, server, "api-key-mailbox-reader")

	apiKey := createAPIKeyPreview(t, state, "api-key-mailbox-reader", portal.CreateAPIKeyInput{
		Name:   "mailbox-reader",
		Scopes: []string{"mailboxes.read"},
	})

	rr := performJSON(server, http.MethodPost, "/api/v1/mailboxes", `{"domainId":1,"expiresInHours":24}`, apiKey)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected api key mailbox create without write scope to return 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAPIKeyDomainBindingRestrictsMailboxCreationToBoundDomain(t *testing.T) {
	server, state := newTestServerWithState(t)
	ownerToken := registerAndLogin(t, server, "api-key-bound-owner")

	rr := performJSON(server, http.MethodPost, "/api/v1/domains", `{"domain":"bound-owned-one.test","status":"active"}`, ownerToken)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected first owned domain create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}
	allowedDomainID := extractJSONScalarField(rr.Body.String(), "id")
	if allowedDomainID == "" {
		t.Fatalf("expected first domain id in response: %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodPost, "/api/v1/domains", `{"domain":"bound-owned-two.test","status":"active"}`, ownerToken)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected second owned domain create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}
	deniedDomainID := extractJSONScalarField(rr.Body.String(), "id")
	if deniedDomainID == "" {
		t.Fatalf("expected second domain id in response: %s", rr.Body.String())
	}

	allowedNodeID := parseUint64(t, allowedDomainID)
	apiKey := createAPIKeyPreview(t, state, "api-key-bound-owner", portal.CreateAPIKeyInput{
		Name:   "bound-mailbox-writer",
		Scopes: []string{"mailboxes.write"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode:         "private_only",
			AllowOwnedPrivateDomains: true,
		},
		DomainBindings: []portal.APIKeyDomainBinding{
			{
				NodeID:      &allowedNodeID,
				AccessLevel: "write",
			},
		},
	})

	rr = performJSON(server, http.MethodPost, "/api/v1/mailboxes", `{"domainId":`+allowedDomainID+`,"expiresInHours":24}`, apiKey)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected mailbox create on bound domain to succeed, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = performJSON(server, http.MethodPost, "/api/v1/mailboxes", `{"domainId":`+deniedDomainID+`,"expiresInHours":24}`, apiKey)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected mailbox create outside bound domain to return 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAPIKeyWithoutBindingsCanCreateMailboxAcrossAllOwnedDomains(t *testing.T) {
	server, state := newTestServerWithState(t)
	ownerToken := registerAndLogin(t, server, "api-key-unbound-owner")

	firstRR := performJSON(server, http.MethodPost, "/api/v1/domains", `{"domain":"unbound-owned-one.test","status":"active"}`, ownerToken)
	if firstRR.Code != http.StatusCreated {
		t.Fatalf("expected first owned domain create to succeed, got %d: %s", firstRR.Code, firstRR.Body.String())
	}
	firstDomainID := extractJSONScalarField(firstRR.Body.String(), "id")
	if firstDomainID == "" {
		t.Fatalf("expected first domain id in response: %s", firstRR.Body.String())
	}

	secondRR := performJSON(server, http.MethodPost, "/api/v1/domains", `{"domain":"unbound-owned-two.test","status":"active"}`, ownerToken)
	if secondRR.Code != http.StatusCreated {
		t.Fatalf("expected second owned domain create to succeed, got %d: %s", secondRR.Code, secondRR.Body.String())
	}
	secondDomainID := extractJSONScalarField(secondRR.Body.String(), "id")
	if secondDomainID == "" {
		t.Fatalf("expected second domain id in response: %s", secondRR.Body.String())
	}

	apiKey := createAPIKeyPreview(t, state, "api-key-unbound-owner", portal.CreateAPIKeyInput{
		Name:   "unbound-mailbox-writer",
		Scopes: []string{"mailboxes.write", "domains.read"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode: "private_only",
		},
	})

	rr := performJSON(server, http.MethodPost, "/api/v1/mailboxes", `{"domainId":`+firstDomainID+`,"expiresInHours":24}`, apiKey)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected mailbox create on first owned domain to succeed without bindings, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = performJSON(server, http.MethodPost, "/api/v1/mailboxes", `{"domainId":`+secondDomainID+`,"expiresInHours":24}`, apiKey)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected mailbox create on second owned domain to succeed without bindings, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/domains", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected unbound api key domain list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"domain":"unbound-owned-one.test"`) || !strings.Contains(rr.Body.String(), `"domain":"unbound-owned-two.test"`) {
		t.Fatalf("expected unbound api key to see all owned domains, got %s", rr.Body.String())
	}
}

func TestAPIKeyWithoutBindingsDoesNotExposeInaccessiblePrivateDomain(t *testing.T) {
	server, state := newTestServerWithState(t)
	adminToken := adminAccessToken(t)
	registerAndLogin(t, server, "api-key-private-owner")
	requesterToken := registerAndLogin(t, server, "api-key-inaccessible-private")

	owner, err := state.AuthRepo.FindUserByLogin(context.Background(), "api-key-private-owner")
	if err != nil {
		t.Fatalf("expected private owner lookup success, got %v", err)
	}

	rr := performJSON(server, http.MethodPost, "/api/v1/admin/domains", `{"domain":"inaccessible-private.test","status":"active","visibility":"private"}`, adminToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin private domain create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}
	domainID := extractJSONScalarField(rr.Body.String(), "id")
	if domainID == "" {
		t.Fatalf("expected private domain id in response: %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodPost, "/api/v1/admin/mailboxes", `{"userId":`+strconv.FormatUint(owner.ID, 10)+`,"domainId":`+domainID+`,"expiresInHours":24}`, adminToken)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected owner mailbox create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}

	if !strings.Contains(rr.Body.String(), `"address":"`) {
		t.Fatalf("expected created owner mailbox in response, got %s", rr.Body.String())
	}

	apiKey := createAPIKeyPreview(t, state, "api-key-inaccessible-private", portal.CreateAPIKeyInput{
		Name:   "unbound-private-check",
		Scopes: []string{"domains.read", "mailboxes.read", "mailboxes.write"},
	})

	rr = performJSON(server, http.MethodGet, "/api/v1/domains", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected api key domain list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"domain":"inaccessible-private.test"`) {
		t.Fatalf("expected inaccessible private domain to stay hidden from unbound api key, got %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/dashboard", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected api key dashboard to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"domain":"inaccessible-private.test"`) {
		t.Fatalf("expected inaccessible private domain to stay hidden from dashboard, got %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"mailboxes":[{`) {
		t.Fatalf("expected dashboard mailbox list to hide inaccessible private mailbox, got %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/mailboxes", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected api key mailbox list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"domain":"inaccessible-private.test"`) {
		t.Fatalf("expected inaccessible private mailbox to stay hidden from mailbox list, got %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodPost, "/api/v1/mailboxes", `{"domainId":`+domainID+`,"expiresInHours":24}`, apiKey)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected mailbox create on inaccessible private domain to return 404, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/domains", "", requesterToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected jwt domain list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"domain":"inaccessible-private.test"`) {
		t.Fatalf("expected inaccessible private domain to stay hidden from jwt domain list too, got %s", rr.Body.String())
	}
}

func TestAdminAPIKeyCanSeeBoundPlatformPrivateDomainAndMailbox(t *testing.T) {
	server, state := newTestServerWithState(t)
	adminToken := adminAccessToken(t)

	adminUser, err := state.AuthRepo.FindUserByLogin(context.Background(), "admin")
	if err != nil {
		t.Fatalf("expected seeded admin user, got %v", err)
	}

	rr := performJSON(server, http.MethodPost, "/api/v1/admin/domains", `{"domain":"admin-platform-private.test","status":"active","visibility":"private"}`, adminToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin private domain create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}
	domainID := parseUint64(t, extractJSONScalarField(rr.Body.String(), "id"))

	rr = performJSON(server, http.MethodPost, "/api/v1/admin/mailboxes", `{"userId":`+strconv.FormatUint(adminUser.ID, 10)+`,"domainId":`+strconv.FormatUint(domainID, 10)+`,"expiresInHours":24}`, adminToken)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected admin mailbox create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}
	mailboxID := extractJSONScalarField(rr.Body.String(), "id")
	if mailboxID == "" {
		t.Fatalf("expected mailbox id in response: %s", rr.Body.String())
	}

	boundNodeID := domainID
	apiKey := createAPIKeyPreview(t, state, "admin", portal.CreateAPIKeyInput{
		Name:   "admin-bound-platform-private",
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

	rr = performJSON(server, http.MethodGet, "/api/v1/domains", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin api key domain list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"domain":"admin-platform-private.test"`) {
		t.Fatalf("expected bound private platform domain in api key domain list: %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/mailboxes", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin api key mailbox list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"id":`+mailboxID) {
		t.Fatalf("expected created mailbox in api key mailbox list: %s", rr.Body.String())
	}
}

func TestAdminAPIKeyWithoutBindingsCanUsePlatformPrivateDomain(t *testing.T) {
	server, state := newTestServerWithState(t)
	adminToken := adminAccessToken(t)

	adminUser, err := state.AuthRepo.FindUserByLogin(context.Background(), "admin")
	if err != nil {
		t.Fatalf("expected seeded admin user, got %v", err)
	}

	rr := performJSON(server, http.MethodPost, "/api/v1/admin/domains", `{"domain":"admin-unbound-platform-private.test","status":"active","visibility":"private"}`, adminToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin private domain create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}
	domainID := extractJSONScalarField(rr.Body.String(), "id")
	if domainID == "" {
		t.Fatalf("expected domain id in response: %s", rr.Body.String())
	}

	apiKey := createAPIKeyPreview(t, state, "admin", portal.CreateAPIKeyInput{
		Name:   "admin-unbound-platform-private",
		Scopes: []string{"domains.read", "mailboxes.read", "mailboxes.write"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode: "private_only",
		},
	})

	rr = performJSON(server, http.MethodGet, "/api/v1/domains", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin api key domain list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"domain":"admin-unbound-platform-private.test"`) {
		t.Fatalf("expected admin api key domain list to include platform private domain, got %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodPost, "/api/v1/mailboxes", `{"domainId":`+domainID+`,"expiresInHours":24}`, apiKey)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected admin api key mailbox create on platform private domain to succeed, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"userId":`+strconv.FormatUint(adminUser.ID, 10)) {
		t.Fatalf("expected created mailbox to belong to admin user, got %s", rr.Body.String())
	}
}

func TestUnboundAPIKeyCanListOwnMailboxOnPlatformPrivateDomain(t *testing.T) {
	server, state := newTestServerWithState(t)
	adminToken := adminAccessToken(t)

	userToken := registerAndLogin(t, server, "api-key-own-platform-mailbox")
	user, err := state.AuthRepo.FindUserByLogin(context.Background(), "api-key-own-platform-mailbox")
	if err != nil {
		t.Fatalf("expected seeded user lookup success, got %v", err)
	}

	rr := performJSON(server, http.MethodPost, "/api/v1/admin/domains", `{"domain":"own-platform-private.test","status":"active","visibility":"private"}`, adminToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin private domain create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}
	domainID := parseUint64(t, extractJSONScalarField(rr.Body.String(), "id"))

	rr = performJSON(server, http.MethodPost, "/api/v1/admin/mailboxes", `{"userId":`+strconv.FormatUint(user.ID, 10)+`,"domainId":`+strconv.FormatUint(domainID, 10)+`,"expiresInHours":24}`, adminToken)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected admin mailbox create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}

	apiKey := createAPIKeyPreview(t, state, "api-key-own-platform-mailbox", portal.CreateAPIKeyInput{
		Name:   "own-platform-mailbox-reader",
		Scopes: []string{"domains.read", "mailboxes.read"},
	})

	rr = performJSON(server, http.MethodGet, "/api/v1/mailboxes", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected mailbox list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"domain":"own-platform-private.test"`) {
		t.Fatalf("expected own mailbox on platform private domain to stay visible, got %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/dashboard", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected dashboard to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"domain":"own-platform-private.test"`) {
		t.Fatalf("expected dashboard mailbox list to keep own platform private mailbox visible, got %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"availableDomains":[{"id":`+strconv.FormatUint(domainID, 10)) {
		t.Fatalf("expected platform private domain to stay unavailable for creation, got %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/domains", "", userToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected jwt domain list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"domain":"own-platform-private.test"`) {
		t.Fatalf("expected platform private domain to stay hidden from standard domain list, got %s", rr.Body.String())
	}
}
