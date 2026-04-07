package tests

import (
	"net/http"
	"testing"

	"shiro-email/backend/internal/modules/portal"
)

func TestAPIKeyCannotCreateMailboxFromSharedDomainWithoutPublicPoolUseScope(t *testing.T) {
	server, state := newTestServerWithState(t)
	adminToken := adminAccessToken(t)
	_ = registerAndLogin(t, server, "api-key-shared-domain-no-use")

	rr := performJSON(server, http.MethodPost, "/api/v1/admin/domains", `{"domain":"shared-mailbox-no-use.test","status":"active","visibility":"platform_public","publicationStatus":"published"}`, adminToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected shared domain create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}
	domainID := extractJSONScalarField(rr.Body.String(), "id")
	if domainID == "" {
		t.Fatalf("expected shared domain id in response: %s", rr.Body.String())
	}

	apiKey := createAPIKeyPreview(t, state, "api-key-shared-domain-no-use", portal.CreateAPIKeyInput{
		Name:   "shared-domain-writer",
		Scopes: []string{"mailboxes.write"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode:           "public_only",
			AllowPlatformPublicDomains: true,
		},
	})

	rr = performJSON(server, http.MethodPost, "/api/v1/mailboxes", `{"domainId":`+domainID+`,"expiresInHours":24}`, apiKey)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected shared domain mailbox create without public_pool.use to return 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAPIKeyCanCreateMailboxFromSharedDomainWithPublicPoolUseScope(t *testing.T) {
	server, state := newTestServerWithState(t)
	adminToken := adminAccessToken(t)
	_ = registerAndLogin(t, server, "api-key-shared-domain-use")

	rr := performJSON(server, http.MethodPost, "/api/v1/admin/domains", `{"domain":"shared-mailbox-use.test","status":"active","visibility":"platform_public","publicationStatus":"published"}`, adminToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected shared domain create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}
	domainID := extractJSONScalarField(rr.Body.String(), "id")
	if domainID == "" {
		t.Fatalf("expected shared domain id in response: %s", rr.Body.String())
	}

	apiKey := createAPIKeyPreview(t, state, "api-key-shared-domain-use", portal.CreateAPIKeyInput{
		Name:   "shared-domain-writer",
		Scopes: []string{"mailboxes.write", "public_pool.use"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode:           "public_only",
			AllowPlatformPublicDomains: true,
		},
	})

	rr = performJSON(server, http.MethodPost, "/api/v1/mailboxes", `{"domainId":`+domainID+`,"expiresInHours":24}`, apiKey)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected shared domain mailbox create with public_pool.use to return 201, got %d: %s", rr.Code, rr.Body.String())
	}
}
