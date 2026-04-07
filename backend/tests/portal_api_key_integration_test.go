package tests

import (
	"net/http"
	"strings"
	"testing"
)

func TestPortalAPIKeyCreateReturnsOneTimePlainSecretAndMaskedListPreview(t *testing.T) {
	server := newTestServer(t)

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"alice","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected alice login to return 200, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")
	if token == "" {
		t.Fatalf("expected alice access token, got %s", login.Body.String())
	}

	createBody := `{"name":"portal-sdk","scopes":["domains.read","mailboxes.read"]}`
	rr := performJSON(server, http.MethodPost, "/api/v1/portal/api-keys", createBody, token)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected portal api key create to return 201, got %d: %s", rr.Code, rr.Body.String())
	}

	plainSecret := extractJSONField(rr.Body.String(), "plainSecret")
	keyPreview := extractJSONField(rr.Body.String(), "keyPreview")
	apiKeyID := extractJSONScalarField(rr.Body.String(), "id")
	if plainSecret == "" || !strings.HasPrefix(plainSecret, "sk_live_") {
		t.Fatalf("expected one-time plain secret in create response: %s", rr.Body.String())
	}
	if keyPreview == "" || keyPreview == plainSecret {
		t.Fatalf("expected masked preview distinct from plain secret, preview=%q secret=%q body=%s", keyPreview, plainSecret, rr.Body.String())
	}
	if apiKeyID == "" {
		t.Fatalf("expected api key id in create response: %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/portal/api-keys", "", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected portal api key list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), plainSecret) {
		t.Fatalf("expected plain secret to stay out of api key list: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), keyPreview) {
		t.Fatalf("expected masked preview to appear in api key list: %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodPost, "/api/v1/portal/api-keys/"+apiKeyID+"/revoke", `{}`, token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected revoke to return 200, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/portal/api-keys", "", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected api key list after revoke to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"name":"portal-sdk"`) {
		t.Fatalf("expected revoked api key to stay hidden from user list: %s", rr.Body.String())
	}
}
