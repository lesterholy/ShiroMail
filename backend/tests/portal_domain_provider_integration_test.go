package tests

import (
	"net/http"
	"strings"
	"testing"
)

func TestUserCanUpdateUnusedOwnedProviderAccountCoreFields(t *testing.T) {
	server := newTestServer(t)

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"alice","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected alice login to return 200, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")
	if token == "" {
		t.Fatalf("expected alice access token, got %s", login.Body.String())
	}

	rr := performJSON(server, http.MethodPost, "/api/v1/portal/domain-providers", `{"provider":"cloudflare","displayName":"User Editable Provider","authType":"api_token","status":"healthy","secretRef":"cf-token","capabilities":["zones.read"]}`, token)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected provider create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = performJSON(server, http.MethodPut, "/api/v1/portal/domain-providers/1", `{"provider":"spaceship","displayName":"User Updated Spaceship","authType":"api_key","credentials":{"apiKey":"ship-key","apiSecret":"ship-secret"},"status":"pending","capabilities":["zones.read","dns.write"]}`, token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected user provider update to succeed, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"provider":"spaceship"`) {
		t.Fatalf("expected provider type to change for unused account: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"authType":"api_key"`) {
		t.Fatalf("expected auth type to change for unused account: %s", rr.Body.String())
	}
}

func TestUserRejectsChangingProviderOrAuthTypeWhenOwnedProviderBound(t *testing.T) {
	server := newTestServer(t)

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"alice","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected alice login to return 200, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")
	if token == "" {
		t.Fatalf("expected alice access token, got %s", login.Body.String())
	}

	rr := performJSON(server, http.MethodPost, "/api/v1/portal/domain-providers", `{"provider":"cloudflare","displayName":"User Bound Provider","authType":"api_token","status":"healthy","secretRef":"cf-token","capabilities":["zones.read","dns.write"]}`, token)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected provider create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = performJSON(server, http.MethodPost, "/api/v1/domains", `{"domain":"bound-owned-provider.test","status":"active","visibility":"private","publicationStatus":"draft","healthStatus":"healthy","providerAccountId":1,"isDefault":false,"weight":100}`, token)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected provider-bound domain create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = performJSON(server, http.MethodPut, "/api/v1/portal/domain-providers/1", `{"provider":"spaceship","displayName":"Should Fail","authType":"api_key","credentials":{"apiKey":"ship-key","apiSecret":"ship-secret"},"status":"pending","capabilities":["zones.read"]}`, token)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when changing core fields on bound user provider, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = performJSON(server, http.MethodPut, "/api/v1/portal/domain-providers/1", `{"provider":"cloudflare","displayName":"Still Bound But Editable","authType":"api_token","credentials":{"apiToken":"cf-token-2"},"status":"degraded","capabilities":["zones.read","dns.read","dns.write"]}`, token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected bound provider to allow safe updates, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"displayName":"Still Bound But Editable"`) {
		t.Fatalf("expected safe field update to persist: %s", rr.Body.String())
	}
}
