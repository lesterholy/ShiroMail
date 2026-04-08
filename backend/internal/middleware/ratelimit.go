package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

func (rl *RateLimiter) Limit(prefix string, maxRequests int, window time.Duration) gin.HandlerFunc {
	return rl.LimitWithKey(prefix, maxRequests, window, func(ctx *gin.Context) string {
		return ctx.ClientIP()
	})
}

func (rl *RateLimiter) LimitDynamic(prefix string, window time.Duration, maxFn func(ctx *gin.Context) int, keyFn func(ctx *gin.Context) string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		maxRequests := 0
		if maxFn != nil {
			maxRequests = maxFn(ctx)
		}
		if maxRequests <= 0 {
			ctx.Next()
			return
		}

		identity := ctx.ClientIP()
		if keyFn != nil {
			if candidate := strings.TrimSpace(keyFn(ctx)); candidate != "" {
				identity = candidate
			}
		}

		key := fmt.Sprintf("ratelimit:%s:%s", prefix, identity)
		count, err := rl.increment(ctx.Request.Context(), key, window)
		if err != nil {
			ctx.Next()
			return
		}

		ctx.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		ctx.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, maxRequests-int(count))))

		if int(count) > maxRequests {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"message": "too many requests, please try again later",
			})
			return
		}

		ctx.Next()
	}
}

func (rl *RateLimiter) LimitWithKey(prefix string, maxRequests int, window time.Duration, keyFn func(ctx *gin.Context) string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identity := ctx.ClientIP()
		if keyFn != nil {
			if candidate := strings.TrimSpace(keyFn(ctx)); candidate != "" {
				identity = candidate
			}
		}
		key := fmt.Sprintf("ratelimit:%s:%s", prefix, identity)
		count, err := rl.increment(ctx.Request.Context(), key, window)
		if err != nil {
			ctx.Next()
			return
		}

		ctx.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		ctx.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, maxRequests-int(count))))

		if int(count) > maxRequests {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"message": "too many requests, please try again later",
			})
			return
		}

		ctx.Next()
	}
}

func RequestIPRateLimitKey(ctx *gin.Context) string {
	return "ip:" + ctx.ClientIP()
}

func RequestHasBearerCredential(ctx *gin.Context) bool {
	header := strings.TrimSpace(ctx.GetHeader("Authorization"))
	if header == "" {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" {
		return false
	}
	return strings.HasPrefix(header, "Bearer ")
}

func RequestIdentityRateLimitKey(ctx *gin.Context) string {
	if !RequestHasBearerCredential(ctx) {
		return RequestIPRateLimitKey(ctx)
	}
	token := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ctx.GetHeader("Authorization")), "Bearer "))
	sum := sha256.Sum256([]byte(token))
	return "bearer:" + hex.EncodeToString(sum[:8])
}

func RequestRateLimitKeyWithMode(mode string) func(ctx *gin.Context) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "ip":
		return RequestIPRateLimitKey
	default:
		return RequestIdentityRateLimitKey
	}
}

func (rl *RateLimiter) increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	pipe := rl.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}
