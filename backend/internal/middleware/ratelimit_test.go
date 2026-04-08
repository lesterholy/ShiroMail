package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestHasBearerCredential(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer sk_live_example")
	ctx.Request = req

	if !RequestHasBearerCredential(ctx) {
		t.Fatal("expected bearer credential to be detected")
	}
}

func TestRequestRateLimitKeyWithModeUsesIPWhenConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.4:12345"
	req.Header.Set("Authorization", "Bearer sk_live_example")
	ctx.Request = req

	key := RequestRateLimitKeyWithMode("ip")(ctx)
	if !strings.HasPrefix(key, "ip:") {
		t.Fatalf("expected ip based key, got %q", key)
	}
}

func TestRequestRateLimitKeyWithModeUsesBearerHashByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.4:12345"
	req.Header.Set("Authorization", "Bearer sk_live_example")
	ctx.Request = req

	key := RequestRateLimitKeyWithMode("bearer_or_ip")(ctx)
	if !strings.HasPrefix(key, "bearer:") {
		t.Fatalf("expected bearer based key, got %q", key)
	}
}
