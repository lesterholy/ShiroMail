package auth

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"shiro-email/backend/internal/modules/system"
	"shiro-email/backend/internal/shared/security"
)

type stubEmailSender struct {
	lastTo      string
	lastCode    string
	lastPurpose string
	lastAction  string
}

func (s *stubEmailSender) SendVerificationCode(_ context.Context, to string, code string, actionURL string) error {
	s.lastTo = to
	s.lastCode = code
	s.lastPurpose = "verification"
	s.lastAction = actionURL
	return nil
}

func (s *stubEmailSender) SendPasswordResetCode(_ context.Context, to string, code string, actionURL string) error {
	s.lastTo = to
	s.lastCode = code
	s.lastPurpose = "password_reset"
	s.lastAction = actionURL
	return nil
}

func TestRegisterAndConfirmEmailVerification(t *testing.T) {
	configRepo := system.NewMemoryConfigRepository()
	_, _ = configRepo.Upsert(context.Background(), system.ConfigKeyAuthRegistrationPolicy, map[string]any{
		"registrationMode":         "public",
		"allowRegistration":        true,
		"requireEmailVerification": true,
		"inviteOnly":               false,
	}, 1)
	_, _ = configRepo.Upsert(context.Background(), system.ConfigKeyMailDelivery, map[string]any{
		"enabled":     true,
		"host":        "smtp.example.com",
		"port":        587,
		"username":    "sender@example.com",
		"password":    "app-password",
		"fromAddress": "sender@example.com",
		"fromName":    "Shiro Email",
	}, 1)

	sender := &stubEmailSender{}
	repo := NewMemoryRepository()
	service := NewService(repo, "secret", configRepo, sender)
	if _, err := repo.CreateUser(context.Background(), User{
		Username:      "bootstrap-admin",
		Email:         "bootstrap-admin@example.com",
		PasswordHash:  mustHashPassword(t, "Secret123!"),
		EmailVerified: true,
		Roles:         []string{"admin", "user"},
	}); err != nil {
		t.Fatalf("seed bootstrap admin: %v", err)
	}

	result, err := service.Register(context.Background(), RegisterRequest{
		Username: "verify-user",
		Email:    "verify-user@example.com",
		Password: "Secret123!",
	})
	if err == nil || result != nil {
		t.Fatal("expected register to require email verification")
	}

	pending, ok := err.(*PendingVerificationError)
	if !ok {
		t.Fatalf("expected pending verification error, got %v", err)
	}
	if pending.Challenge.VerificationTicket == "" {
		t.Fatal("expected verification ticket")
	}
	if sender.lastTo != "verify-user@example.com" || sender.lastCode == "" {
		t.Fatalf("expected verification email to be sent, got to=%q code=%q", sender.lastTo, sender.lastCode)
	}
	if sender.lastPurpose != "verification" {
		t.Fatalf("expected verification purpose, got %q", sender.lastPurpose)
	}
	if sender.lastAction == "" {
		t.Fatal("expected verification action url")
	}

	authResult, confirmErr := service.ConfirmEmailVerification(context.Background(), EmailVerificationConfirmRequest{
		VerificationTicket: pending.Challenge.VerificationTicket,
		Code:               sender.lastCode,
	})
	if confirmErr != nil {
		t.Fatalf("confirm verification: %v", confirmErr)
	}
	if authResult.AccessToken == "" || authResult.RefreshToken == "" {
		t.Fatalf("expected issued tokens after verification, got %+v", authResult)
	}
}

func TestRegisterBootstrapsFirstAdminWhenNoAdminExists(t *testing.T) {
	configRepo := system.NewMemoryConfigRepository()
	_, _ = configRepo.Upsert(context.Background(), system.ConfigKeyAuthRegistrationPolicy, map[string]any{
		"registrationMode":         "closed",
		"allowRegistration":        false,
		"requireEmailVerification": true,
		"inviteOnly":               true,
	}, 1)

	repo := NewMemoryRepository()
	service := NewService(repo, "secret", configRepo, &stubEmailSender{})

	result, err := service.Register(context.Background(), RegisterRequest{
		Username: "first-admin",
		Email:    "first-admin@example.com",
		Password: "Secret123!",
	})
	if err != nil {
		t.Fatalf("bootstrap register should succeed: %v", err)
	}
	if result == nil {
		t.Fatal("expected issued session for bootstrap admin")
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatalf("expected issued tokens, got %+v", result)
	}
	if !slices.Contains(result.Roles, "admin") {
		t.Fatalf("expected admin role, got %+v", result.Roles)
	}
}

func TestSettingsReportsBootstrapAdminRequirementUntilFirstAdminExists(t *testing.T) {
	configRepo := system.NewMemoryConfigRepository()
	repo := NewMemoryRepository()
	service := NewService(repo, "secret", configRepo, &stubEmailSender{})
	ctx := context.Background()

	settings, err := service.Settings(ctx)
	if err != nil {
		t.Fatalf("load settings without admin: %v", err)
	}
	if !settings.BootstrapAdminRequired {
		t.Fatalf("expected bootstrap admin to be required, got %+v", settings)
	}

	if _, err := repo.CreateUser(ctx, User{
		Username:      "admin-user",
		Email:         "admin-user@example.com",
		PasswordHash:  mustHashPassword(t, "Secret123!"),
		EmailVerified: true,
		Roles:         []string{"admin", "user"},
	}); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	settings, err = service.Settings(ctx)
	if err != nil {
		t.Fatalf("load settings with admin: %v", err)
	}
	if settings.BootstrapAdminRequired {
		t.Fatalf("expected bootstrap admin requirement to clear, got %+v", settings)
	}
}

func TestForgotAndResetPasswordWithVerificationCode(t *testing.T) {
	configRepo := system.NewMemoryConfigRepository()
	_, _ = configRepo.Upsert(context.Background(), system.ConfigKeyMailDelivery, map[string]any{
		"enabled":     true,
		"host":        "smtp.example.com",
		"port":        587,
		"username":    "sender@example.com",
		"password":    "app-password",
		"fromAddress": "sender@example.com",
		"fromName":    "Shiro Email",
	}, 1)

	sender := &stubEmailSender{}
	service := NewService(NewMemoryRepository(), "secret", configRepo, sender)

	_, err := service.Register(context.Background(), RegisterRequest{
		Username: "reset-user",
		Email:    "reset-user@example.com",
		Password: "Secret123!",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	forgotResult, err := service.ForgotPassword(context.Background(), ForgotPasswordRequest{
		Login: "reset-user@example.com",
	})
	if err != nil {
		t.Fatalf("forgot password: %v", err)
	}
	if forgotResult.VerificationTicket == "" {
		t.Fatal("expected verification ticket")
	}
	if forgotResult.Email != "reset-user@example.com" {
		t.Fatalf("expected reset email, got %q", forgotResult.Email)
	}
	if sender.lastTo != "reset-user@example.com" || sender.lastCode == "" {
		t.Fatalf("expected password reset email, got to=%q code=%q", sender.lastTo, sender.lastCode)
	}
	if sender.lastPurpose != "password_reset" {
		t.Fatalf("expected password reset purpose, got %q", sender.lastPurpose)
	}
	if sender.lastAction == "" {
		t.Fatal("expected password reset action url")
	}

	_, err = service.ResetPassword(context.Background(), ResetPasswordRequest{
		VerificationTicket: forgotResult.VerificationTicket,
		Code:               sender.lastCode,
		NewPassword:        "BetterSecret456!",
	})
	if err != nil {
		t.Fatalf("reset password: %v", err)
	}

	_, err = service.Login(context.Background(), LoginRequest{
		Login:    "reset-user@example.com",
		Password: "BetterSecret456!",
	})
	if err != nil {
		t.Fatalf("login with new password: %v", err)
	}
}

func TestResendPasswordResetUsesPasswordResetDelivery(t *testing.T) {
	configRepo := system.NewMemoryConfigRepository()
	_, _ = configRepo.Upsert(context.Background(), system.ConfigKeyMailDelivery, map[string]any{
		"enabled":     true,
		"host":        "smtp.example.com",
		"port":        587,
		"username":    "sender@example.com",
		"password":    "app-password",
		"fromAddress": "sender@example.com",
		"fromName":    "Shiro Email",
	}, 1)

	sender := &stubEmailSender{}
	service := NewService(NewMemoryRepository(), "secret", configRepo, sender)

	_, err := service.Register(context.Background(), RegisterRequest{
		Username: "resend-user",
		Email:    "resend-user@example.com",
		Password: "Secret123!",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	forgotResult, err := service.ForgotPassword(context.Background(), ForgotPasswordRequest{
		Login: "resend-user@example.com",
	})
	if err != nil {
		t.Fatalf("forgot password: %v", err)
	}

	sender.lastPurpose = ""
	sender.lastCode = ""

	result, err := service.ResendEmailVerification(context.Background(), EmailVerificationResendRequest{
		VerificationTicket: forgotResult.VerificationTicket,
	})
	if err != nil {
		t.Fatalf("resend password reset: %v", err)
	}
	if result.VerificationTicket == "" {
		t.Fatal("expected a new verification ticket")
	}
	if sender.lastPurpose != "password_reset" {
		t.Fatalf("expected password reset resend purpose, got %q", sender.lastPurpose)
	}
	if sender.lastCode == "" {
		t.Fatal("expected password reset resend code")
	}
	if sender.lastAction == "" {
		t.Fatal("expected password reset resend action url")
	}
}

func TestRegisterRequiresMailDeliveryBeforePersistingPendingUser(t *testing.T) {
	configRepo := system.NewMemoryConfigRepository()
	_, _ = configRepo.Upsert(context.Background(), system.ConfigKeyAuthRegistrationPolicy, map[string]any{
		"registrationMode":         "public",
		"allowRegistration":        true,
		"requireEmailVerification": true,
		"inviteOnly":               false,
	}, 1)

	repo := NewMemoryRepository()
	service := NewService(repo, "secret", configRepo, &stubEmailSender{})
	ctx := context.Background()

	if _, err := repo.CreateUser(ctx, User{
		Username:      "bootstrap-admin",
		Email:         "bootstrap-admin@example.com",
		PasswordHash:  mustHashPassword(t, "Secret123!"),
		EmailVerified: true,
		Roles:         []string{"admin", "user"},
	}); err != nil {
		t.Fatalf("seed bootstrap admin: %v", err)
	}

	result, err := service.Register(ctx, RegisterRequest{
		Username: "mail-disabled-user",
		Email:    "mail-disabled@example.com",
		Password: "Secret123!",
	})
	if result != nil {
		t.Fatalf("expected no auth result, got %+v", result)
	}
	if err == nil || err.Error() != "mail delivery is disabled" {
		t.Fatalf("expected mail delivery disabled error, got %v", err)
	}

	if _, lookupErr := repo.FindUserByLogin(ctx, "mail-disabled@example.com"); !errors.Is(lookupErr, ErrUserNotFound) {
		t.Fatalf("expected no residual pending user, got %v", lookupErr)
	}
}

func TestRegisterReusesExistingPendingUserAfterMailDeliveryRecovery(t *testing.T) {
	configRepo := system.NewMemoryConfigRepository()
	_, _ = configRepo.Upsert(context.Background(), system.ConfigKeyAuthRegistrationPolicy, map[string]any{
		"registrationMode":         "public",
		"allowRegistration":        true,
		"requireEmailVerification": true,
		"inviteOnly":               false,
	}, 1)
	_, _ = configRepo.Upsert(context.Background(), system.ConfigKeyMailDelivery, map[string]any{
		"enabled":     true,
		"host":        "smtp.example.com",
		"port":        587,
		"username":    "sender@example.com",
		"password":    "app-password",
		"fromAddress": "sender@example.com",
		"fromName":    "Shiro Email",
	}, 1)

	sender := &stubEmailSender{}
	repo := NewMemoryRepository()
	service := NewService(repo, "secret", configRepo, sender)
	ctx := context.Background()

	if _, err := repo.CreateUser(ctx, User{
		Username:      "bootstrap-admin",
		Email:         "bootstrap-admin@example.com",
		PasswordHash:  mustHashPassword(t, "Secret123!"),
		EmailVerified: true,
		Roles:         []string{"admin", "user"},
	}); err != nil {
		t.Fatalf("seed bootstrap admin: %v", err)
	}

	legacyUser, err := repo.CreateUser(ctx, User{
		Username:      "stale-user",
		Email:         "recover@example.com",
		PasswordHash:  mustHashPassword(t, "OldSecret123!"),
		Status:        "pending_verification",
		EmailVerified: false,
		Roles:         []string{"user"},
	})
	if err != nil {
		t.Fatalf("seed stale pending user: %v", err)
	}

	result, registerErr := service.Register(ctx, RegisterRequest{
		Username: "fresh-user",
		Email:    "recover@example.com",
		Password: "NewSecret123!",
	})
	if registerErr == nil || result != nil {
		t.Fatal("expected pending verification response")
	}

	pending, ok := registerErr.(*PendingVerificationError)
	if !ok {
		t.Fatalf("expected pending verification error, got %v", registerErr)
	}
	if pending.Challenge.VerificationTicket == "" {
		t.Fatal("expected verification ticket")
	}
	if sender.lastTo != "recover@example.com" || sender.lastCode == "" {
		t.Fatalf("expected verification email to be resent, got to=%q code=%q", sender.lastTo, sender.lastCode)
	}

	user, err := repo.FindUserByLogin(ctx, "recover@example.com")
	if err != nil {
		t.Fatalf("find recovered user: %v", err)
	}
	if user.ID != legacyUser.ID {
		t.Fatalf("expected to reuse pending user %d, got %d", legacyUser.ID, user.ID)
	}
	if user.Username != "fresh-user" {
		t.Fatalf("expected pending username to refresh, got %q", user.Username)
	}
	if user.Status != "pending_verification" || user.EmailVerified {
		t.Fatalf("expected recovered user to stay pending, got %+v", user)
	}
	if !security.VerifyPassword(user.PasswordHash, "NewSecret123!") {
		t.Fatal("expected pending user password to refresh")
	}
}

func TestMemoryRepositoryStoresTOTPCredentialAndMFAChallenge(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	err := repo.UpsertTOTPCredential(ctx, TOTPCredential{
		UserID:           7,
		SecretCiphertext: "secret",
		Enabled:          false,
	})
	if err != nil {
		t.Fatalf("upsert totp credential: %v", err)
	}

	credential, err := repo.FindTOTPCredentialByUserID(ctx, 7)
	if err != nil {
		t.Fatalf("find totp credential: %v", err)
	}
	if credential.SecretCiphertext != "secret" {
		t.Fatalf("expected stored secret, got %+v", credential)
	}

	challenge, err := repo.SaveMFAChallenge(ctx, MFAChallengeRecord{
		UserID:     7,
		TicketHash: "ticket-hash",
		Purpose:    "login_totp",
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("save challenge: %v", err)
	}
	if challenge.ID == 0 {
		t.Fatal("expected challenge id")
	}
}

func TestServiceCanUpdateAccountProfileAndPassword(t *testing.T) {
	configRepo := system.NewMemoryConfigRepository()
	service := NewService(NewMemoryRepository(), "secret", configRepo, &stubEmailSender{})
	ctx := context.Background()

	user, err := service.repo.CreateUser(ctx, User{
		Username:     "profile-user",
		Email:        "profile@example.com",
		PasswordHash: mustHashPassword(t, "Secret123!"),
		Roles:        []string{"user"},
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	updated, err := service.UpdateAccountProfile(ctx, user.ID, UpdateAccountProfileRequest{
		DisplayName:        "Profile User",
		Locale:             "zh-CN",
		Timezone:           "Asia/Shanghai",
		AutoRefreshSeconds: 45,
	})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.DisplayName != "Profile User" {
		t.Fatalf("expected updated display name, got %+v", updated)
	}

	err = service.ChangePassword(ctx, user.ID, ChangePasswordRequest{
		CurrentPassword: "Secret123!",
		NewPassword:     "BetterSecret456!",
	})
	if err != nil {
		t.Fatalf("change password: %v", err)
	}

	_, err = service.Login(ctx, LoginRequest{
		Login:    "profile@example.com",
		Password: "BetterSecret456!",
	})
	if err != nil {
		t.Fatalf("login with new password: %v", err)
	}
}

func TestGetAccountProfileRepairsLegacyPlaceholderDisplayName(t *testing.T) {
	configRepo := system.NewMemoryConfigRepository()
	repo := NewMemoryRepository()
	service := NewService(repo, "secret", configRepo, &stubEmailSender{})
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, User{
		Username:      "legacy-user",
		Email:         "legacy-user@example.com",
		PasswordHash:  mustHashPassword(t, "Secret123!"),
		EmailVerified: true,
		Roles:         []string{"user"},
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if _, err := repo.UpsertProfileSettings(ctx, ProfileSettings{
		UserID:             user.ID,
		DisplayName:        "GALA Workspace",
		Locale:             "zh-CN",
		Timezone:           "Asia/Shanghai",
		AutoRefreshSeconds: 30,
	}); err != nil {
		t.Fatalf("seed legacy profile: %v", err)
	}

	profile, err := service.GetAccountProfile(ctx, user.ID)
	if err != nil {
		t.Fatalf("get account profile: %v", err)
	}
	if profile.DisplayName != "legacy-user" {
		t.Fatalf("expected repaired display name, got %+v", profile)
	}
}

func TestServiceRequiresVerificationForBoundEmailChange(t *testing.T) {
	configRepo := system.NewMemoryConfigRepository()
	_, _ = configRepo.Upsert(context.Background(), system.ConfigKeyMailDelivery, map[string]any{
		"enabled":     true,
		"host":        "smtp.example.com",
		"port":        587,
		"username":    "sender@example.com",
		"password":    "app-password",
		"fromAddress": "sender@example.com",
		"fromName":    "Shiro Email",
	}, 1)

	sender := &stubEmailSender{}
	repo := NewMemoryRepository()
	service := NewService(repo, "secret", configRepo, sender)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, User{
		Username:      "email-user",
		Email:         "old@example.com",
		PasswordHash:  mustHashPassword(t, "Secret123!"),
		EmailVerified: true,
		Roles:         []string{"user"},
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	challenge, err := service.RequestEmailChange(ctx, user.ID, RequestEmailChangeRequest{
		NewEmail: "new@example.com",
	})
	if err != nil {
		t.Fatalf("request email change: %v", err)
	}
	if challenge.VerificationTicket == "" {
		t.Fatal("expected verification ticket")
	}
	if sender.lastAction == "" {
		t.Fatal("expected email change action url")
	}
	if want := "/dashboard/settings?action=change-email"; !contains(sender.lastAction, want) {
		t.Fatalf("expected email change action url to include %q, got %q", want, sender.lastAction)
	}

	updated, err := service.ConfirmEmailChange(ctx, user.ID, ConfirmEmailChangeRequest{
		VerificationTicket: challenge.VerificationTicket,
		Code:               sender.lastCode,
	})
	if err != nil {
		t.Fatalf("confirm email change: %v", err)
	}
	if updated.Email != "new@example.com" {
		t.Fatalf("expected updated email, got %+v", updated)
	}
}

func contains(value string, expected string) bool {
	return strings.Contains(value, expected)
}

func TestLoginReturnsTwoFactorRequiredWhenTOTPEnabled(t *testing.T) {
	configRepo := system.NewMemoryConfigRepository()
	repo := NewMemoryRepository()
	service := NewService(repo, "secret", configRepo, &stubEmailSender{})
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, User{
		Username:      "mfa-user",
		Email:         "mfa@example.com",
		PasswordHash:  mustHashPassword(t, "Secret123!"),
		EmailVerified: true,
		Roles:         []string{"user"},
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	setup, err := service.SetupTOTP(ctx, user.ID)
	if err != nil {
		t.Fatalf("setup totp: %v", err)
	}
	code, err := generateTOTPCodeAt(setup.ManualEntryKey, time.Now())
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}
	if err := service.EnableTOTP(ctx, user.ID, EnableTOTPRequest{Code: code}); err != nil {
		t.Fatalf("enable totp: %v", err)
	}

	result, err := service.Login(ctx, LoginRequest{Login: "mfa-user", Password: "Secret123!"})
	if err == nil || result != nil {
		t.Fatal("expected two-factor challenge instead of session")
	}

	challenge, ok := err.(*PendingMFAError)
	if !ok || challenge.Challenge.ChallengeTicket == "" {
		t.Fatalf("expected mfa challenge, got %v", err)
	}
}

func TestVerifyLoginTOTPConsumesChallengeAndIssuesTokens(t *testing.T) {
	configRepo := system.NewMemoryConfigRepository()
	repo := NewMemoryRepository()
	service := NewService(repo, "secret", configRepo, &stubEmailSender{})
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, User{
		Username:      "verify-mfa",
		Email:         "verify-mfa@example.com",
		PasswordHash:  mustHashPassword(t, "Secret123!"),
		EmailVerified: true,
		Roles:         []string{"user"},
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	setup, err := service.SetupTOTP(ctx, user.ID)
	if err != nil {
		t.Fatalf("setup totp: %v", err)
	}
	code, err := generateTOTPCodeAt(setup.ManualEntryKey, time.Now())
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}
	if err := service.EnableTOTP(ctx, user.ID, EnableTOTPRequest{Code: code}); err != nil {
		t.Fatalf("enable totp: %v", err)
	}

	_, loginErr := service.Login(ctx, LoginRequest{Login: "verify-mfa", Password: "Secret123!"})
	challenge, ok := loginErr.(*PendingMFAError)
	if !ok {
		t.Fatalf("expected pending mfa error, got %v", loginErr)
	}

	verifyCode, err := generateTOTPCodeAt(setup.ManualEntryKey, time.Now())
	if err != nil {
		t.Fatalf("generate verify code: %v", err)
	}
	session, err := service.VerifyLoginTOTP(ctx, VerifyLoginTOTPRequest{
		ChallengeTicket: challenge.Challenge.ChallengeTicket,
		Code:            verifyCode,
	})
	if err != nil {
		t.Fatalf("verify login totp: %v", err)
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatalf("expected issued tokens, got %+v", session)
	}
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()

	hash, err := security.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return hash
}
