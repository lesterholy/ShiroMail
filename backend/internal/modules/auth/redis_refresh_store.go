package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisRefreshStore struct {
	client *redis.Client
}

func NewRedisRefreshStore(client *redis.Client) *RedisRefreshStore {
	return &RedisRefreshStore{client: client}
}

func (s *RedisRefreshStore) SaveRefreshToken(ctx context.Context, record RefreshTokenRecord) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}

	ttl := time.Until(record.ExpiresAt)
	if ttl <= 0 {
		ttl = time.Second
	}
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, s.key(record.TokenHash), payload, ttl)
	pipe.SAdd(ctx, s.userKey(record.UserID), record.TokenHash)
	pipe.Expire(ctx, s.userKey(record.UserID), ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisRefreshStore) FindRefreshToken(ctx context.Context, tokenHash string) (RefreshTokenRecord, error) {
	raw, err := s.client.Get(ctx, s.key(tokenHash)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return RefreshTokenRecord{}, ErrRefreshTokenNotFound
		}
		return RefreshTokenRecord{}, err
	}

	var record RefreshTokenRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return RefreshTokenRecord{}, err
	}
	return record, nil
}

func (s *RedisRefreshStore) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	record, err := s.FindRefreshToken(ctx, tokenHash)
	if err != nil {
		return err
	}

	deleted, err := s.client.Del(ctx, s.key(tokenHash)).Result()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrRefreshTokenNotFound
	}
	_ = s.client.SRem(ctx, s.userKey(record.UserID), tokenHash).Err()
	return nil
}

func (s *RedisRefreshStore) RevokeUserRefreshTokens(ctx context.Context, userID uint64) error {
	hashes, err := s.client.SMembers(ctx, s.userKey(userID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	if len(hashes) == 0 {
		return nil
	}

	keys := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		keys = append(keys, s.key(hash))
	}
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, keys...)
	pipe.Del(ctx, s.userKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisRefreshStore) key(tokenHash string) string {
	return "auth:refresh:" + tokenHash
}

func (s *RedisRefreshStore) userKey(userID uint64) string {
	return "auth:refresh:user:" + fmt.Sprintf("%d", userID)
}
