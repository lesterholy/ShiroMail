package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"shiro-email/backend/internal/modules/system"
	"shiro-email/backend/internal/shared/security"
)

type oauthStateRecord struct {
	Provider     string
	CodeVerifier string
	ExpiresAt    time.Time
}

type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type oauthProfile struct {
	Subject  string
	Email    string
	Username string
}

func (s *Service) StartOAuth(ctx context.Context, provider string) (*OAuthStartResponse, error) {
	settings, err := system.LoadAuthRuntimeSettings(ctx, s.configRepo)
	if err != nil {
		return nil, err
	}

	config, ok := settings.OAuth[provider]
	if !ok {
		return nil, errors.New("oauth provider not found")
	}
	if !config.Enabled {
		return nil, errors.New("oauth provider disabled")
	}
	if strings.TrimSpace(config.ClientID) == "" || strings.TrimSpace(config.RedirectURL) == "" || strings.TrimSpace(config.AuthorizationURL) == "" {
		return nil, errors.New("oauth provider is not fully configured")
	}

	state, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	codeVerifier, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(10 * time.Minute)
	s.stateMu.Lock()
	if s.oauthByState == nil {
		s.oauthByState = make(map[string]oauthStateRecord)
	}
	s.oauthByState[state] = oauthStateRecord{
		Provider:     provider,
		CodeVerifier: codeVerifier,
		ExpiresAt:    expiresAt,
	}
	s.stateMu.Unlock()

	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", config.ClientID)
	values.Set("redirect_uri", config.RedirectURL)
	values.Set("scope", strings.Join(config.Scopes, " "))
	values.Set("state", state)
	if config.UsePKCE {
		values.Set("code_challenge", oauthCodeChallenge(codeVerifier))
		values.Set("code_challenge_method", "S256")
	}

	return &OAuthStartResponse{
		Provider:         provider,
		AuthorizationURL: config.AuthorizationURL + "?" + values.Encode(),
	}, nil
}

func (s *Service) CompleteOAuth(ctx context.Context, provider string, req OAuthCallbackRequest) (*AuthResponse, error) {
	settings, err := system.LoadAuthRuntimeSettings(ctx, s.configRepo)
	if err != nil {
		return nil, err
	}

	config, ok := settings.OAuth[provider]
	if !ok {
		return nil, errors.New("oauth provider not found")
	}
	if !config.Enabled {
		return nil, errors.New("oauth provider disabled")
	}

	s.stateMu.Lock()
	stateRecord, ok := s.oauthByState[req.State]
	if ok && time.Now().After(stateRecord.ExpiresAt) {
		delete(s.oauthByState, req.State)
		ok = false
	}
	if ok {
		delete(s.oauthByState, req.State)
	}
	s.stateMu.Unlock()

	if !ok || stateRecord.Provider != provider {
		return nil, errors.New("invalid oauth state")
	}

	accessToken, err := exchangeOAuthCode(ctx, config, req.Code, stateRecord.CodeVerifier)
	if err != nil {
		return nil, err
	}

	profile, err := fetchOAuthProfile(ctx, provider, config, accessToken)
	if err != nil {
		return nil, err
	}

	user, err := s.resolveOAuthUser(ctx, provider, config, profile)
	if err != nil {
		return nil, err
	}
	if settings.Registration.RequireEmailVerification && !user.EmailVerified {
		pending, challengeErr := s.issueVerificationChallenge(ctx, user, "oauth")
		if challengeErr != nil {
			return nil, challengeErr
		}
		return nil, pending
	}
	return s.issueTokens(ctx, user)
}

func exchangeOAuthCode(ctx context.Context, config system.AuthOAuthProviderConfig, code string, codeVerifier string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", config.RedirectURL)
	form.Set("client_id", config.ClientID)
	if strings.TrimSpace(config.ClientSecret) != "" {
		form.Set("client_secret", config.ClientSecret)
	}
	if config.UsePKCE {
		form.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("oauth token exchange failed: %s", strings.TrimSpace(string(body)))
	}

	var token oauthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", err
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return "", errors.New("oauth token exchange returned empty access token")
	}
	return token.AccessToken, nil
}

func fetchOAuthProfile(ctx context.Context, provider string, config system.AuthOAuthProviderConfig, accessToken string) (oauthProfile, error) {
	switch provider {
	case "google":
		return fetchGoogleProfile(ctx, config.UserInfoURL, accessToken)
	case "github":
		return fetchGitHubProfile(ctx, config.UserInfoURL, accessToken)
	case "microsoft":
		return fetchMicrosoftProfile(ctx, config.UserInfoURL, accessToken)
	default:
		return fetchGenericOIDCProfile(ctx, provider, config.UserInfoURL, accessToken)
	}
}

func fetchGoogleProfile(ctx context.Context, userInfoURL string, accessToken string) (oauthProfile, error) {
	var payload struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := fetchBearerJSON(ctx, userInfoURL, accessToken, &payload); err != nil {
		return oauthProfile{}, err
	}
	return oauthProfile{
		Subject:  payload.Sub,
		Email:    strings.TrimSpace(payload.Email),
		Username: normalizeOAuthUsername(payload.Name, payload.Email, "google-user"),
	}, nil
}

func fetchMicrosoftProfile(ctx context.Context, userInfoURL string, accessToken string) (oauthProfile, error) {
	var payload struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
	}
	if err := fetchBearerJSON(ctx, userInfoURL, accessToken, &payload); err != nil {
		return oauthProfile{}, err
	}
	email := strings.TrimSpace(payload.Email)
	if email == "" {
		email = strings.TrimSpace(payload.PreferredUsername)
	}
	return oauthProfile{
		Subject:  payload.Sub,
		Email:    email,
		Username: normalizeOAuthUsername(payload.Name, email, "microsoft-user"),
	}, nil
}

func fetchGitHubProfile(ctx context.Context, userInfoURL string, accessToken string) (oauthProfile, error) {
	var payload struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := fetchBearerJSON(ctx, userInfoURL, accessToken, &payload); err != nil {
		return oauthProfile{}, err
	}
	email := strings.TrimSpace(payload.Email)
	if email == "" {
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := fetchBearerJSON(ctx, "https://api.github.com/user/emails", accessToken, &emails); err == nil {
			for _, item := range emails {
				if item.Primary {
					email = strings.TrimSpace(item.Email)
					break
				}
			}
			if email == "" && len(emails) > 0 {
				email = strings.TrimSpace(emails[0].Email)
			}
		}
	}
	return oauthProfile{
		Subject:  fmt.Sprintf("%d", payload.ID),
		Email:    email,
		Username: normalizeOAuthUsername(payload.Login, email, "github-user"),
	}, nil
}

func fetchBearerJSON(ctx context.Context, endpoint string, accessToken string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("oauth user info failed: %s", strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (s *Service) resolveOAuthUser(ctx context.Context, provider string, config system.AuthOAuthProviderConfig, profile oauthProfile) (User, error) {
	email := strings.TrimSpace(profile.Email)
	if email == "" {
		email = fmt.Sprintf("%s-%s@oauth.shiro.local", provider, strings.TrimSpace(profile.Subject))
	}

	existing, err := s.repo.FindUserByLogin(ctx, email)
	if err == nil {
		if !config.AllowLinkExisting {
			return User{}, errors.New("oauth provider cannot link existing account")
		}
		return existing, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return User{}, err
	}
	if !config.AllowAutoRegister {
		return User{}, errors.New("oauth auto registration is disabled")
	}

	passwordHash, err := security.HashPassword(provider + "-" + profile.Subject + "-" + time.Now().Format(time.RFC3339Nano))
	if err != nil {
		return User{}, err
	}

	baseUsername := normalizeOAuthUsername(profile.Username, email, provider+"-user")
	for attempt := 0; attempt < 20; attempt++ {
		username := baseUsername
		if attempt > 0 {
			username = fmt.Sprintf("%s-%d", baseUsername, attempt+1)
		}

		created, createErr := s.repo.CreateUser(ctx, User{
			Username:      username,
			Email:         email,
			PasswordHash:  passwordHash,
			Status:        "pending_verification",
			EmailVerified: false,
			Roles:         []string{"user"},
		})
		if createErr == nil {
			return created, nil
		}
		if !strings.Contains(strings.ToLower(createErr.Error()), "username already exists") {
			return User{}, createErr
		}
	}
	return User{}, errors.New("failed to allocate oauth username")
}

func oauthCodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func normalizeOAuthUsername(primary string, fallback string, defaultValue string) string {
	candidate := strings.TrimSpace(primary)
	if candidate == "" {
		candidate = strings.TrimSpace(fallback)
	}
	if candidate == "" {
		candidate = defaultValue
	}
	candidate = strings.ToLower(candidate)
	replacer := strings.NewReplacer("@", "-", ".", "-", "_", "-", " ", "-")
	candidate = replacer.Replace(candidate)
	candidate = strings.Trim(candidate, "-")
	if candidate == "" {
		return defaultValue
	}
	return candidate
}

func fetchGenericOIDCProfile(ctx context.Context, provider string, userInfoURL string, accessToken string) (oauthProfile, error) {
	if strings.TrimSpace(userInfoURL) == "" {
		return oauthProfile{}, errors.New("oauth provider has no userInfoUrl configured")
	}
	var payload struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
		Login             string `json:"login"`
		ID                int64  `json:"id"`
	}
	if err := fetchBearerJSON(ctx, userInfoURL, accessToken, &payload); err != nil {
		return oauthProfile{}, err
	}
	subject := strings.TrimSpace(payload.Sub)
	if subject == "" && payload.ID != 0 {
		subject = fmt.Sprintf("%d", payload.ID)
	}
	email := strings.TrimSpace(payload.Email)
	if email == "" {
		email = strings.TrimSpace(payload.PreferredUsername)
	}
	username := strings.TrimSpace(payload.Name)
	if username == "" {
		username = strings.TrimSpace(payload.Login)
	}
	return oauthProfile{
		Subject:  subject,
		Email:    email,
		Username: normalizeOAuthUsername(username, email, provider+"-user"),
	}, nil
}
