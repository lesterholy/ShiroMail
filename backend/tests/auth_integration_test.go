package tests

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"shiro-email/backend/internal/bootstrap"
)

func performJSON(server http.Handler, method string, path string, body string, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	return rr
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	return bootstrap.NewRouterForTest()
}

func TestRegisterAndLoginFlow(t *testing.T) {
	server := newTestServer(t)

	registerBody := `{"username":"reggie","email":"reggie@example.com","password":"Secret123!"}`
	rr := performJSON(server, http.MethodPost, "/api/v1/auth/register", registerBody, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	loginBody := `{"login":"reggie","password":"Secret123!"}`
	rr = performJSON(server, http.MethodPost, "/api/v1/auth/login", loginBody, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "accessToken") {
		t.Fatal("expected access token in response")
	}
}

func TestSeededDemoAccountsCanLogin(t *testing.T) {
	server := newTestServer(t)

	aliceBody := `{"login":"alice","password":"Secret123!"}`
	rr := performJSON(server, http.MethodPost, "/api/v1/auth/login", aliceBody, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected seeded alice login to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"username":"alice"`) {
		t.Fatalf("expected alice identity in response: %s", rr.Body.String())
	}

	adminBody := `{"login":"admin","password":"Secret123!"}`
	rr = performJSON(server, http.MethodPost, "/api/v1/auth/login", adminBody, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected seeded admin login to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"roles":[`) || !strings.Contains(rr.Body.String(), `"admin"`) {
		t.Fatalf("expected admin role in response: %s", rr.Body.String())
	}
}

func TestAuthEndpointsAllowFrontendCORS(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "content-type,authorization")

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for CORS preflight, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:5173" {
		t.Fatalf("expected frontend origin to be allowed, got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("expected Access-Control-Allow-Methods header")
	}
	if rr.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Fatal("expected Access-Control-Allow-Headers header")
	}
}

func TestAuthEndpointsAllowLocalhostFrontendCORS(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "content-type,authorization")

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for CORS preflight, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("expected localhost frontend origin to be allowed, got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("expected Access-Control-Allow-Methods header")
	}
	if rr.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Fatal("expected Access-Control-Allow-Headers header")
	}
}

func TestRefreshAndLogoutFlow(t *testing.T) {
	server := newTestServer(t)

	registerBody := `{"username":"bob","email":"bob@example.com","password":"Secret123!"}`
	rr := performJSON(server, http.MethodPost, "/api/v1/auth/register", registerBody, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	loginBody := `{"login":"bob","password":"Secret123!"}`
	rr = performJSON(server, http.MethodPost, "/api/v1/auth/login", loginBody, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	refreshToken := extractJSONField(body, "refreshToken")
	accessToken := extractJSONField(body, "accessToken")
	if refreshToken == "" || accessToken == "" {
		t.Fatalf("expected tokens in login response: %s", body)
	}

	refreshBody := `{"refreshToken":"` + refreshToken + `"}`
	rr = performJSON(server, http.MethodPost, "/api/v1/auth/refresh", refreshBody, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on refresh, got %d: %s", rr.Code, rr.Body.String())
	}

	logoutBody := `{"refreshToken":"` + refreshToken + `"}`
	rr = performJSON(server, http.MethodPost, "/api/v1/auth/logout", logoutBody, accessToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on logout, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = performJSON(server, http.MethodPost, "/api/v1/auth/refresh", refreshBody, "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAuthSettingsEndpointExposesStructuredRuntimeSettings(t *testing.T) {
	server := newTestServer(t)

	rr := performJSON(server, http.MethodGet, "/api/v1/auth/settings", "", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"allowRegistration":true`) {
		t.Fatalf("expected allowRegistration in auth settings response: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"oauthProviders"`) {
		t.Fatalf("expected oauthProviders in auth settings response: %s", rr.Body.String())
	}
}

func TestPublicSiteSettingsEndpointExposesStructuredSiteConfig(t *testing.T) {
	server := newTestServer(t)

	rr := performJSON(server, http.MethodGet, "/api/v1/site/settings", "", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"identity"`) {
		t.Fatalf("expected identity in public site settings response: %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"branding"`) || strings.Contains(rr.Body.String(), `"landing"`) {
		t.Fatalf("expected removed branding/landing fields to stay absent: %s", rr.Body.String())
	}
}

func TestForgotAndResetPasswordFlow(t *testing.T) {
	server := newTestServer(t)

	registerBody := `{"username":"reset-user","email":"reset-user@example.com","password":"Secret123!"}`
	rr := performJSON(server, http.MethodPost, "/api/v1/auth/register", registerBody, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	forgotBody := `{"login":"reset-user"}`
	rr = performJSON(server, http.MethodPost, "/api/v1/auth/forgot-password", forgotBody, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	verificationTicket := extractJSONField(rr.Body.String(), "verificationTicket")
	if verificationTicket == "" {
		t.Fatalf("expected verification ticket in forgot password response: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"email":"reset-user@example.com"`) {
		t.Fatalf("expected reset email in response: %s", rr.Body.String())
	}

	resetBody := `{"verificationTicket":"` + verificationTicket + `","code":"000000","newPassword":"BetterSecret456!"}`
	rr = performJSON(server, http.MethodPost, "/api/v1/auth/reset-password", resetBody, "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid code, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "invalid verification code") {
		t.Fatalf("expected invalid verification code message, got: %s", rr.Body.String())
	}
}

func TestRegisterRespectsRegistrationPolicyConfig(t *testing.T) {
	server := newTestServer(t)

	adminLogin := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"admin","password":"Secret123!"}`, "")
	if adminLogin.Code != http.StatusOK {
		t.Fatalf("expected admin login success, got %d: %s", adminLogin.Code, adminLogin.Body.String())
	}
	adminToken := extractJSONField(adminLogin.Body.String(), "accessToken")
	if adminToken == "" {
		t.Fatalf("expected admin access token: %s", adminLogin.Body.String())
	}

	configBody := `{"value":{"registrationMode":"closed","allowRegistration":false,"requireEmailVerification":false,"inviteOnly":false}}`
	rr := performJSON(server, http.MethodPut, "/api/v1/admin/configs/auth.registration_policy", configBody, adminToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected config upsert success, got %d: %s", rr.Code, rr.Body.String())
	}

	registerBody := `{"username":"blocked","email":"blocked@example.com","password":"Secret123!"}`
	rr = performJSON(server, http.MethodPost, "/api/v1/auth/register", registerBody, "")
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 when registration disabled, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAccountProfileAndPasswordEndpoints(t *testing.T) {
	server := newTestServer(t)

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"alice","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")

	rr := performJSON(server, http.MethodPatch, "/api/v1/account/profile", `{"displayName":"Alice Ops","locale":"en-US","timezone":"UTC","autoRefreshSeconds":60}`, token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"displayName":"Alice Ops"`) {
		t.Fatalf("expected updated display name, got %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodPost, "/api/v1/account/password/change", `{"currentPassword":"Secret123!","newPassword":"BetterSecret456!"}`, token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected password change success, got %d: %s", rr.Code, rr.Body.String())
	}

	relogin := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"alice","password":"BetterSecret456!"}`, "")
	if relogin.Code != http.StatusOK {
		t.Fatalf("expected login with new password success, got %d: %s", relogin.Code, relogin.Body.String())
	}
}

func TestLoginReturnsTwoFactorRequiredWhenEnabled(t *testing.T) {
	server := newTestServer(t)

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"alice","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected initial login success, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")

	setup := performJSON(server, http.MethodPost, "/api/v1/account/2fa/totp/setup", `{}`, token)
	if setup.Code != http.StatusOK {
		t.Fatalf("expected setup success, got %d: %s", setup.Code, setup.Body.String())
	}
	manualKey := extractJSONField(setup.Body.String(), "manualEntryKey")
	code, err := generateTestTOTPCode(manualKey)
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}

	enable := performJSON(server, http.MethodPost, "/api/v1/account/2fa/totp/enable", `{"code":"`+code+`"}`, token)
	if enable.Code != http.StatusOK {
		t.Fatalf("expected enable success, got %d: %s", enable.Code, enable.Body.String())
	}

	relogin := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"alice","password":"Secret123!"}`, "")
	if relogin.Code != http.StatusForbidden {
		t.Fatalf("expected two factor challenge, got %d: %s", relogin.Code, relogin.Body.String())
	}
	if !strings.Contains(relogin.Body.String(), `"status":"two_factor_required"`) {
		t.Fatalf("expected two factor status, got %s", relogin.Body.String())
	}
}

func TestDisableTOTPTurnsOffTwoFactor(t *testing.T) {
	server := newTestServer(t)

	login := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"alice","password":"Secret123!"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("expected initial login success, got %d: %s", login.Code, login.Body.String())
	}
	token := extractJSONField(login.Body.String(), "accessToken")

	setup := performJSON(server, http.MethodPost, "/api/v1/account/2fa/totp/setup", `{}`, token)
	if setup.Code != http.StatusOK {
		t.Fatalf("expected setup success, got %d: %s", setup.Code, setup.Body.String())
	}
	manualKey := extractJSONField(setup.Body.String(), "manualEntryKey")
	code, err := generateTestTOTPCode(manualKey)
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}

	enable := performJSON(server, http.MethodPost, "/api/v1/account/2fa/totp/enable", `{"code":"`+code+`"}`, token)
	if enable.Code != http.StatusOK {
		t.Fatalf("expected enable success, got %d: %s", enable.Code, enable.Body.String())
	}

	disable := performJSON(server, http.MethodPost, "/api/v1/account/2fa/totp/disable", `{"password":"Secret123!"}`, token)
	if disable.Code != http.StatusOK {
		t.Fatalf("expected disable success, got %d: %s", disable.Code, disable.Body.String())
	}

	status := performJSON(server, http.MethodGet, "/api/v1/account/2fa/status", "", token)
	if status.Code != http.StatusOK {
		t.Fatalf("expected status success, got %d: %s", status.Code, status.Body.String())
	}
	if !strings.Contains(status.Body.String(), `"enabled":false`) {
		t.Fatalf("expected two factor disabled in status response, got %s", status.Body.String())
	}

	profile := performJSON(server, http.MethodGet, "/api/v1/account/profile", "", token)
	if profile.Code != http.StatusOK {
		t.Fatalf("expected profile success, got %d: %s", profile.Code, profile.Body.String())
	}
	if !strings.Contains(profile.Body.String(), `"twoFactorEnabled":false`) {
		t.Fatalf("expected profile to report twoFactorEnabled false, got %s", profile.Body.String())
	}

	relogin := performJSON(server, http.MethodPost, "/api/v1/auth/login", `{"login":"alice","password":"Secret123!"}`, "")
	if relogin.Code != http.StatusOK {
		t.Fatalf("expected login without two factor after disable, got %d: %s", relogin.Code, relogin.Body.String())
	}
}

func TestNewlyRegisteredUserGetsPortalDefaults(t *testing.T) {
	server := newTestServer(t)

	registerBody := `{"username":"fresh-user","email":"fresh-user@example.com","password":"Secret123!"}`
	register := performJSON(server, http.MethodPost, "/api/v1/auth/register", registerBody, "")
	if register.Code != http.StatusCreated {
		t.Fatalf("expected 201 on register, got %d: %s", register.Code, register.Body.String())
	}
	token := extractJSONField(register.Body.String(), "accessToken")
	if token == "" {
		t.Fatalf("expected access token in register response: %s", register.Body.String())
	}

	dashboard := performJSON(server, http.MethodGet, "/api/v1/dashboard", "", token)
	if dashboard.Code != http.StatusOK {
		t.Fatalf("expected dashboard success, got %d: %s", dashboard.Code, dashboard.Body.String())
	}

	overview := performJSON(server, http.MethodGet, "/api/v1/portal/overview", "", token)
	if overview.Code != http.StatusOK {
		t.Fatalf("expected portal overview success, got %d: %s", overview.Code, overview.Body.String())
	}
	if !strings.Contains(overview.Body.String(), `"username":"fresh-user"`) {
		t.Fatalf("expected portal overview to include the new user, got %s", overview.Body.String())
	}
	if strings.Contains(overview.Body.String(), `GALA Workspace`) {
		t.Fatalf("expected overview to avoid legacy placeholder display name, got %s", overview.Body.String())
	}

	billing := performJSON(server, http.MethodGet, "/api/v1/portal/billing", "", token)
	if billing.Code != http.StatusOK {
		t.Fatalf("expected billing success, got %d: %s", billing.Code, billing.Body.String())
	}
	if !strings.Contains(billing.Body.String(), `"planCode":"free"`) {
		t.Fatalf("expected default billing profile, got %s", billing.Body.String())
	}

	profile := performJSON(server, http.MethodGet, "/api/v1/account/profile", "", token)
	if profile.Code != http.StatusOK {
		t.Fatalf("expected account profile success, got %d: %s", profile.Code, profile.Body.String())
	}
	if !strings.Contains(profile.Body.String(), `"displayName":"fresh-user"`) {
		t.Fatalf("expected username-based displayName for new users, got %s", profile.Body.String())
	}
}

func extractJSONField(body string, key string) string {
	needle := `"` + key + `":"`
	start := strings.Index(body, needle)
	if start == -1 {
		return ""
	}
	start += len(needle)
	end := strings.Index(body[start:], `"`)
	if end == -1 {
		return ""
	}
	return body[start : start+end]
}

func generateTestTOTPCode(secret string) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", err
	}

	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(time.Now().Unix()/30))
	mac := hmac.New(sha1.New, key)
	if _, err := mac.Write(msg[:]); err != nil {
		return "", err
	}
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := (int(sum[offset])&0x7f)<<24 |
		(int(sum[offset+1])&0xff)<<16 |
		(int(sum[offset+2])&0xff)<<8 |
		(int(sum[offset+3]) & 0xff)
	return fmt.Sprintf("%06d", code%1000000), nil
}
