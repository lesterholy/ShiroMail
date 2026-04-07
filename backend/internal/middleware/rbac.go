package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRoles(allowed ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		value, exists := ctx.Get("auth.roles")
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			return
		}
		roles, ok := value.([]string)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			return
		}
		for _, role := range roles {
			for _, allow := range allowed {
				if role == allow {
					ctx.Next()
					return
				}
			}
		}
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "forbidden"})
	}
}
