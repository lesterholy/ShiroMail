package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAllowBrowserClientsExposesDownloadHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(AllowBrowserClients("http://127.0.0.1:5173", "http://localhost:5173"))
	engine.GET("/download", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/download", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	rr := httptest.NewRecorder()

	engine.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5173" {
		t.Fatalf("expected allow origin header, got %q", got)
	}

	expected := "Content-Disposition, Content-Length, Content-Type"
	if got := rr.Header().Get("Access-Control-Expose-Headers"); got != expected {
		t.Fatalf("expected exposed headers %q, got %q", expected, got)
	}
}

func TestAllowBrowserClientsAllowsLocalhostDevOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(AllowBrowserClients("http://127.0.0.1:5173", "http://localhost:5173"))
	engine.GET("/download", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/download", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")

	rr := httptest.NewRecorder()
	engine.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for localhost preflight, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected localhost origin to be allowed, got %q", got)
	}
}

func TestAllowBrowserClientsRejectsUnknownOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(AllowBrowserClients("http://127.0.0.1:5173"))
	engine.GET("/data", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rr := httptest.NewRecorder()

	engine.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow origin header for unknown origin, got %q", got)
	}
}
