package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"shiro-email/backend/internal/modules/auth"
	"shiro-email/backend/internal/modules/portal"
)

type stubAPIKeyAuthenticator struct {
	key portal.APIKey
	err error
}

func (s stubAPIKeyAuthenticator) AuthenticateAPIKey(_ context.Context, _ string) (portal.APIKey, error) {
	if s.err != nil {
		return portal.APIKey{}, s.err
	}
	return s.key, nil
}

type stubUserRoleLookup struct {
	user auth.User
	err  error
}

func (s stubUserRoleLookup) FindUserByID(_ context.Context, _ uint64) (auth.User, error) {
	if s.err != nil {
		return auth.User{}, s.err
	}
	return s.user, nil
}

func TestRequireUserOrAPIKeyUsesStoredUserRolesForAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET(
		"/admin-check",
		RequireUserOrAPIKey("secret", stubAPIKeyAuthenticator{key: portal.APIKey{UserID: 7}}, stubUserRoleLookup{user: auth.User{ID: 7, Roles: []string{"admin", "user"}}}),
		RequireRoles("admin"),
		func(ctx *gin.Context) {
			ctx.Status(http.StatusNoContent)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/admin-check", nil)
	req.Header.Set("Authorization", "Bearer sk_live_test")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected admin api key request success, got %d", rr.Code)
	}
}

func TestRequireUserOrAPIKeyFallsBackWhenRoleLookupFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET(
		"/admin-check",
		RequireUserOrAPIKey("secret", stubAPIKeyAuthenticator{key: portal.APIKey{UserID: 7}}, stubUserRoleLookup{err: errors.New("lookup failed")}),
		RequireRoles("admin"),
		func(ctx *gin.Context) {
			ctx.Status(http.StatusNoContent)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/admin-check", nil)
	req.Header.Set("Authorization", "Bearer sk_live_test")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected role fallback to deny admin route, got %d", rr.Code)
	}
}
