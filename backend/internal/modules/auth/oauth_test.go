package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"shiro-email/backend/internal/modules/system"
)

func TestOAuthStartAndCompleteWithPKCE(t *testing.T) {
	var seenVerifier string

	providerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse token form: %v", err)
			}
			seenVerifier = r.Form.Get("code_verifier")
			if r.Form.Get("grant_type") != "authorization_code" {
				t.Fatalf("expected authorization_code grant, got %q", r.Form.Get("grant_type"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "oauth-access-token",
				"token_type":   "Bearer",
			})
		case "/userinfo":
			if got := r.Header.Get("Authorization"); got != "Bearer oauth-access-token" {
				t.Fatalf("expected bearer token, got %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sub":   "google-user-1",
				"email": "oauth-user@example.com",
				"name":  "OAuth User",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer providerServer.Close()

	configRepo := system.NewMemoryConfigRepository()
	_, _ = configRepo.Upsert(context.Background(), system.ConfigKeyAuthOAuthGoogle, map[string]any{
		"enabled":           true,
		"clientId":          "client-id",
		"clientSecret":      "client-secret",
		"redirectUrl":       "http://localhost:5173/auth/callback/google",
		"authorizationUrl":  providerServer.URL + "/authorize",
		"tokenUrl":          providerServer.URL + "/token",
		"userInfoUrl":       providerServer.URL + "/userinfo",
		"scopes":            []any{"openid", "email", "profile"},
		"usePkce":           true,
		"allowAutoRegister": true,
		"allowLinkExisting": true,
		"displayName":       "Google",
	}, 1)

	repo := NewMemoryRepository()
	service := NewService(repo, "secret", configRepo)

	started, err := service.StartOAuth(context.Background(), "google")
	if err != nil {
		t.Fatalf("start oauth: %v", err)
	}

	parsedURL, err := url.Parse(started.AuthorizationURL)
	if err != nil {
		t.Fatalf("parse authorization url: %v", err)
	}
	values := parsedURL.Query()
	if values.Get("code_challenge_method") != "S256" {
		t.Fatalf("expected S256 challenge method, got %q", values.Get("code_challenge_method"))
	}
	state := values.Get("state")
	if state == "" {
		t.Fatal("expected oauth state in authorization url")
	}

	result, err := service.CompleteOAuth(context.Background(), "google", OAuthCallbackRequest{
		Code:  "provider-code",
		State: state,
	})
	if err != nil {
		t.Fatalf("complete oauth: %v", err)
	}
	if seenVerifier == "" {
		t.Fatal("expected code verifier to be forwarded to token exchange")
	}
	if result.Username == "" || !strings.Contains(strings.Join(result.Roles, ","), "user") {
		t.Fatalf("expected oauth login result with user role, got %+v", result)
	}
}
