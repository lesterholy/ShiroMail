package bootstrap

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"shiro-email/backend/internal/modules/system"
)

const apiLimitsRefreshInterval = 2 * time.Second

type apiLimitsRuntimeCache struct {
	repo       system.ConfigRepository
	refreshTTL time.Duration

	mu          sync.RWMutex
	cached      system.APILimitsConfig
	cachedAt    time.Time
	initialized bool
}

func newAPILimitsRuntimeCache(repo system.ConfigRepository) *apiLimitsRuntimeCache {
	return &apiLimitsRuntimeCache{
		repo:       repo,
		refreshTTL: apiLimitsRefreshInterval,
	}
}

func (c *apiLimitsRuntimeCache) Current(ctx context.Context) system.APILimitsConfig {
	now := time.Now()

	c.mu.RLock()
	if c.initialized && now.Sub(c.cachedAt) < c.refreshTTL {
		current := c.cached
		c.mu.RUnlock()
		return current
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	now = time.Now()
	if c.initialized && now.Sub(c.cachedAt) < c.refreshTTL {
		return c.cached
	}

	current, err := system.LoadAPILimitsSettings(ctx, c.repo)
	if err != nil {
		if c.initialized {
			slog.Warn("api rate limit config refresh failed, keep cached settings", "error", err)
			return c.cached
		}
		slog.Warn("api rate limit config initial load failed, fallback to defaults", "error", err)
		current = resolveAPILimitsRuntimeSettings(ctx, c.repo)
	}

	c.cached = current
	c.cachedAt = now
	c.initialized = true
	return current
}
