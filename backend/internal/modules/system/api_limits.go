package system

import "context"

func LoadAPILimitsSettings(ctx context.Context, repo ConfigRepository) (APILimitsConfig, error) {
	if repo == nil {
		item := NormalizeConfigEntryForTest(ConfigEntry{Key: ConfigKeyAPILimits, Value: map[string]any{}})
		return apiLimitsConfigFromEntry(item), nil
	}

	items, err := repo.List(ctx)
	if err != nil {
		return APILimitsConfig{}, err
	}
	for _, item := range items {
		if item.Key == ConfigKeyAPILimits {
			return apiLimitsConfigFromEntry(NormalizeConfigEntryForTest(item)), nil
		}
	}

	item := NormalizeConfigEntryForTest(ConfigEntry{Key: ConfigKeyAPILimits, Value: map[string]any{}})
	return apiLimitsConfigFromEntry(item), nil
}

func apiLimitsConfigFromEntry(item ConfigEntry) APILimitsConfig {
	return APILimitsConfig{
		Enabled:                     item.Value["enabled"].(bool),
		IdentityMode:                item.Value["identityMode"].(string),
		AnonymousRPM:                item.Value["anonymousRPM"].(int),
		AuthenticatedRPM:            item.Value["authenticatedRPM"].(int),
		AuthRPM:                     item.Value["authRPM"].(int),
		LoginRPM:                    item.Value["loginRPM"].(int),
		RegisterRPM:                 item.Value["registerRPM"].(int),
		RefreshRPM:                  item.Value["refreshRPM"].(int),
		ForgotPasswordRPM:           item.Value["forgotPasswordRPM"].(int),
		ResetPasswordRPM:            item.Value["resetPasswordRPM"].(int),
		EmailVerificationResendRPM:  item.Value["emailVerificationResendRPM"].(int),
		EmailVerificationConfirmRPM: item.Value["emailVerificationConfirmRPM"].(int),
		OAuthStartRPM:               item.Value["oauthStartRPM"].(int),
		OAuthCallbackRPM:            item.Value["oauthCallbackRPM"].(int),
		Login2FAVerifyRPM:           item.Value["login2faVerifyRPM"].(int),
		MailboxWriteRPM:             item.Value["mailboxWriteRPM"].(int),
		StrictIPEnabled:             item.Value["strictIpEnabled"].(bool),
		StrictIPRPM:                 item.Value["strictIpRPM"].(int),
	}
}
