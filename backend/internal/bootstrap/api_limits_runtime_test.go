package bootstrap

import (
	"context"
	"testing"
	"time"

	"shiro-email/backend/internal/modules/system"
)

func TestAPILimitsRuntimeCacheRefreshesUpdatedConfig(t *testing.T) {
	repo := system.NewMemoryConfigRepository()
	cache := newAPILimitsRuntimeCache(repo)
	cache.refreshTTL = 10 * time.Millisecond

	initial := cache.Current(context.Background())
	if initial.AnonymousRPM != 120 || initial.AuthenticatedRPM != 600 {
		t.Fatalf("unexpected initial defaults: %+v", initial)
	}

	if _, err := repo.Upsert(context.Background(), system.ConfigKeyAPILimits, map[string]any{
		"enabled":                     true,
		"identityMode":                "ip",
		"anonymousRPM":                240,
		"authenticatedRPM":            900,
		"authRPM":                     20,
		"loginRPM":                    20,
		"registerRPM":                 20,
		"refreshRPM":                  50,
		"forgotPasswordRPM":           15,
		"resetPasswordRPM":            15,
		"emailVerificationResendRPM":  25,
		"emailVerificationConfirmRPM": 35,
		"oauthStartRPM":               40,
		"oauthCallbackRPM":            40,
		"login2faVerifyRPM":           30,
		"mailboxWriteRPM":             1500,
		"strictIpEnabled":             true,
		"strictIpRPM":                 1600,
	}, 1); err != nil {
		t.Fatalf("upsert api limits: %v", err)
	}

	stillCached := cache.Current(context.Background())
	if stillCached.AnonymousRPM != initial.AnonymousRPM {
		t.Fatalf("expected cached value before ttl expires, got %+v", stillCached)
	}

	time.Sleep(20 * time.Millisecond)

	updated := cache.Current(context.Background())
	if updated.IdentityMode != "ip" || updated.AnonymousRPM != 240 || !updated.StrictIPEnabled {
		t.Fatalf("expected refreshed config after ttl, got %+v", updated)
	}
	if updated.EmailVerificationResendRPM != 25 || updated.OAuthStartRPM != 40 || updated.Login2FAVerifyRPM != 30 {
		t.Fatalf("expected refreshed config after ttl, got %+v", updated)
	}
}
