package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	ErrTokenNotFound = errors.New("token not found")
	logger           = logrus.New()
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
		tokens:   make(map[int]timeoutToken),
		timeout:  timeout,
		timeouts: make(chan int),
	}
	go store.tidy() // background process to handle ttl
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
	tokens   map[int]timeoutToken
	timeout  time.Duration
	mu       sync.RWMutex
	timeouts chan int
}

func (ms *memoryTokenStore) Set(ctx context.Context, id int, token string) error {
	ms.mu.Lock()
	ms.tokens[id] = timeoutToken{
		expires: time.Now().Add(ms.timeout),
		token:   token,
	}
	ms.mu.Unlock()
	timer := time.NewTimer(ms.timeout)
	go func() {
		defer timer.Stop()
		<-timer.C
		ms.timeouts <- id
	}()
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

func (ms *memoryTokenStore) tidy() {
	ctx := context.Background()
	for {
		id, ok := <-ms.timeouts
		if !ok {
			return
		}
		err := ms.Del(ctx, id)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err,
				"id":    id,
			}).Warn("could not clear from in-memory token store")
		}
	}
}
