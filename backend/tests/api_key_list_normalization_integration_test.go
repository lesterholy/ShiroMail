package tests

import (
	"net/http"
	"strings"
	"testing"
)

func TestPortalAPIKeyListReturnsArrayBindingsInsteadOfNull(t *testing.T) {
	server := newTestServer(t)
	rr := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"alice","password":"Secret123!"}`, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected alice login to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	token := extractJSONField(rr.Body.String(), "accessToken")
	if token == "" {
		t.Fatalf("expected alice access token, got %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/portal/api-keys", "", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected portal api key list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"domainBindings":null`) {
		t.Fatalf("expected portal api key list to normalize null bindings: %s", rr.Body.String())
	}
}

func TestAdminAPIKeyListReturnsArrayBindingsInsteadOfNull(t *testing.T) {
	server := newTestServer(t)
	rr := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"admin","password":"Secret123!"}`, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin login to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	token := extractJSONField(rr.Body.String(), "accessToken")
	if token == "" {
		t.Fatalf("expected admin access token, got %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/admin/api-keys", "", token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin api key list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"domainBindings":null`) {
		t.Fatalf("expected admin api key list to normalize null bindings: %s", rr.Body.String())
	}
}
