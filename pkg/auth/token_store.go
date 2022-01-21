package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

var (
	ErrTokenNotFound = errors.New("token not found")
)

type TokenStore interface {
	Set(ctx context.Context, id int, token string) error
	Get(ctx context.Context, id int) (string, error)
	Del(ctx context.Context, id int) error
}

func NewRedisTokenStore(timeout time.Duration, client *redis.Client) TokenStore {
	return &redisTokenStore{client: client, timeout: timeout}
}

func NewInMemoryTokenStore(timeout time.Duration) TokenStore {
	store := &memoryTokenStore{
		tokens:  make(map[int]timeoutToken),
		timeout: timeout,
	}
	// TODO start a background process that will periodically clean out expired
	// tokens.
	return store
}

type redisTokenStore struct {
	client  *redis.Client
	timeout time.Duration
}

func (rs *redisTokenStore) Set(ctx context.Context, id int, token string) error {
	key := fmt.Sprintf("rt:%d", id)
	return rs.client.Set(ctx, key, token, rs.timeout).Err()
}

func (rs *redisTokenStore) Get(ctx context.Context, id int) (string, error) {
	key := fmt.Sprintf("rt:%d", id)
	res, err := rs.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrTokenNotFound
	}
	return res, nil
}

func (rs *redisTokenStore) Del(ctx context.Context, id int) error {
	key := fmt.Sprintf("rt:%d", id)
	return rs.client.Del(ctx, key).Err()
}

type timeoutToken struct {
	expires time.Time
	token   string
}

type memoryTokenStore struct {
	tokens  map[int]timeoutToken
	timeout time.Duration
	mu      sync.RWMutex
}

func (ms *memoryTokenStore) Set(ctx context.Context, id int, token string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.tokens[id] = timeoutToken{
		expires: time.Now().Add(ms.timeout),
		token:   token,
	}
	return nil
}

func (ms *memoryTokenStore) Get(ctx context.Context, id int) (string, error) {
	ms.mu.RLock()
	tmTok, ok := ms.tokens[id]
	if !ok {
		ms.mu.RUnlock()
		return "", ErrTokenNotFound
	}
	ms.mu.RUnlock()
	if time.Now().After(tmTok.expires) {
		ms.Del(ctx, id)
		return "", ErrTokenNotFound
	}
	return tmTok.token, nil
}

func (ms *memoryTokenStore) Del(ctx context.Context, id int) error {
	ms.mu.Lock()
	delete(ms.tokens, id)
	ms.mu.Unlock()
	return nil
}
