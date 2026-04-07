package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

type JSONCache struct {
	client *redis.Client
}

func NewJSONCache(client *redis.Client) *JSONCache {
	return &JSONCache{client: client}
}

func (c *JSONCache) Get(ctx context.Context, key string, dest any) (bool, error) {
	if c == nil || c.client == nil {
		return false, nil
	}

	body, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal(body, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (c *JSONCache) Set(ctx context.Context, key string, ttl time.Duration, value any) error {
	if c == nil || c.client == nil {
		return nil
	}

	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, body, ttl).Err()
}

func (c *JSONCache) Delete(ctx context.Context, keys ...string) error {
	if c == nil || c.client == nil || len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

func (c *JSONCache) DeleteByPattern(ctx context.Context, pattern string) error {
	if c == nil || c.client == nil || pattern == "" {
		return nil
	}

	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) != 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			return nil
		}
	}
}
