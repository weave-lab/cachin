package persist

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis"
)

// RedisStore is a Store that uses redis to store cache data
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new RedisStore.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

// Get searches for a key that matches the provided key in the redis cache. If the key does not exist
// no error will be returned
func (s *RedisStore) Get(_ context.Context, key string) ([]byte, time.Time, error) {
	cmd := s.client.Get(SafeKey(key))
	raw, err := cmd.Bytes()
	switch {
	case errors.Is(err, redis.Nil):
		return nil, time.Time{}, nil
	case err != nil:
		return nil, time.Time{}, err
	}

	d := rawData{}
	err = json.Unmarshal(raw, &d)
	if err != nil {
		return nil, time.Time{}, err
	}

	return d.Raw, d.LastSet, nil
}

// Set updates the redis cache, if the key can't be updated or created an error will
// be returned
func (s *RedisStore) Set(_ context.Context, key string, val []byte) error {
	d, err := json.Marshal(rawData{LastSet: time.Now(), Raw: val})
	if err != nil {
		return err
	}

	cmd := s.client.Set(SafeKey(key), d, Forever)
	return cmd.Err()
}
