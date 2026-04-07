package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AllowBrowserClients(allowedOrigins ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if origin != "" && allowed[origin] {
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Vary", "Origin")
			ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			ctx.Header("Access-Control-Expose-Headers", "Content-Disposition, Content-Length, Content-Type")
			ctx.Header("Access-Control-Allow-Credentials", "true")
			ctx.Header("Access-Control-Max-Age", "3600")

			requestHeaders := ctx.GetHeader("Access-Control-Request-Headers")
			if requestHeaders == "" {
				requestHeaders = "Authorization, Content-Type"
			}
			ctx.Header("Access-Control-Allow-Headers", requestHeaders)
		}

		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
