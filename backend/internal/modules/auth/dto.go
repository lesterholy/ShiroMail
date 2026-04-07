package auth

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type ForgotPasswordRequest struct {
	Login string `json:"login" binding:"required"`
}

type ForgotPasswordResponse struct {
	Status             string `json:"status"`
	Email              string `json:"email"`
	VerificationTicket string `json:"verificationTicket"`
	ExpiresInSeconds   int    `json:"expiresInSeconds"`
}

type ResetPasswordRequest struct {
	VerificationTicket string `json:"verificationTicket" binding:"required"`
	Code               string `json:"code" binding:"required"`
	NewPassword        string `json:"newPassword" binding:"required"`
}

type ResetPasswordResponse struct {
	Status string `json:"status"`
}

type OAuthStartResponse struct {
	Provider         string `json:"provider"`
	AuthorizationURL string `json:"authorizationUrl"`
}

type OAuthCallbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

type EmailVerificationConfirmRequest struct {
	VerificationTicket string `json:"verificationTicket" binding:"required"`
	Code               string `json:"code" binding:"required"`
}

type EmailVerificationResendRequest struct {
	VerificationTicket string `json:"verificationTicket" binding:"required"`
}

type AuthResponse struct {
	Status       string   `json:"status"`
	UserID       uint64   `json:"userId"`
	Username     string   `json:"username"`
	Roles        []string `json:"roles"`
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
}

type AccountProfileResponse struct {
	UserID             uint64   `json:"userId"`
	Username           string   `json:"username"`
	Email              string   `json:"email"`
	EmailVerified      bool     `json:"emailVerified"`
	Roles              []string `json:"roles"`
	DisplayName        string   `json:"displayName"`
	Locale             string   `json:"locale"`
	Timezone           string   `json:"timezone"`
	AutoRefreshSeconds int      `json:"autoRefreshSeconds"`
	TwoFactorEnabled   bool     `json:"twoFactorEnabled"`
}

type UpdateAccountProfileRequest struct {
	DisplayName        string `json:"displayName"`
	Locale             string `json:"locale"`
	Timezone           string `json:"timezone"`
	AutoRefreshSeconds int    `json:"autoRefreshSeconds"`
}

type RequestEmailChangeRequest struct {
	NewEmail string `json:"newEmail" binding:"required"`
}

type ConfirmEmailChangeRequest struct {
	VerificationTicket string `json:"verificationTicket" binding:"required"`
	Code               string `json:"code" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required"`
}

type TOTPStatusResponse struct {
	Enabled bool `json:"enabled"`
}

type SetupTOTPResponse struct {
	ManualEntryKey string `json:"manualEntryKey"`
	OTPAuthURL     string `json:"otpauthUrl"`
}

type EnableTOTPRequest struct {
	Code string `json:"code" binding:"required"`
}

type DisableTOTPRequest struct {
	Password string `json:"password" binding:"required"`
}

type MFALoginChallengeResponse struct {
	Status           string `json:"status"`
	ChallengeTicket  string `json:"challengeTicket"`
	ExpiresInSeconds int    `json:"expiresInSeconds"`
}

type VerifyLoginTOTPRequest struct {
	ChallengeTicket string `json:"challengeTicket" binding:"required"`
	Code            string `json:"code" binding:"required"`
}

type VerificationChallengeResponse struct {
	Status             string `json:"status"`
	Email              string `json:"email"`
	VerificationTicket string `json:"verificationTicket"`
	ExpiresInSeconds   int    `json:"expiresInSeconds"`
}

type AuthSettingsResponse struct {
	RegistrationMode         string                          `json:"registrationMode"`
	AllowRegistration        bool                            `json:"allowRegistration"`
	BootstrapAdminRequired   bool                            `json:"bootstrapAdminRequired"`
	RequireEmailVerification bool                            `json:"requireEmailVerification"`
	InviteOnly               bool                            `json:"inviteOnly"`
	PasswordMinLength        int                             `json:"passwordMinLength"`
	AllowMultiSession        bool                            `json:"allowMultiSession"`
	RefreshTokenDays         int                             `json:"refreshTokenDays"`
	OAuthShowOnLogin         bool                            `json:"oauthShowOnLogin"`
	OAuthProviders           map[string]OAuthProviderSummary `json:"oauthProviders"`
}

type OAuthProviderSummary struct {
	Enabled           bool     `json:"enabled"`
	DisplayName       string   `json:"displayName"`
	RedirectURL       string   `json:"redirectUrl"`
	AuthorizationURL  string   `json:"authorizationUrl"`
	Scopes            []string `json:"scopes"`
	UsePKCE           bool     `json:"usePkce"`
	AllowAutoRegister bool     `json:"allowAutoRegister"`
}
