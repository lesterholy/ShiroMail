package middleware

import (
	"context"
	"fmt"
	"net/http"
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
	return func(ctx *gin.Context) {
		key := fmt.Sprintf("ratelimit:%s:%s", prefix, ctx.ClientIP())
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

func (rl *RateLimiter) increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	pipe := rl.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}
