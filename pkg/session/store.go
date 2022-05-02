package session

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RegisterSerializable will register a type for most session store's default
// serialization.
func RegisterSerializable(t any) {
	gob.Register(t)
}

type Setter[T any] interface {
	Set(ctx context.Context, key string, val *T) error
}

type Getter[T any] interface {
	Get(ctx context.Context, key string) (*T, error)
}

type Deleter interface {
	Del(ctx context.Context, key string) error
}

type Store[T any] interface {
	Setter[T]
	Getter[T]
	Deleter
}

var ErrSessionNotFound = errors.New("session not found")

func NewRedisStore[T any](client redis.UniversalClient, ttl time.Duration) *RedisStore[T] {
	return &RedisStore[T]{c: client, ttl: ttl}
}

func NewMemStore[T any](ttl time.Duration) *MemStore[T] {
	s := MemStore[T]{
		m:   make(map[string]*memstoreValue[T]),
		ttl: ttl,
	}
	go s.tidy(time.Second)
	return &s
}

type RedisStore[T any] struct {
	c   redis.UniversalClient
	ttl time.Duration
}

func (rs *RedisStore[T]) Set(ctx context.Context, key string, val *T) error {
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(val)
	if err != nil {
		return err
	}
	return rs.c.Set(ctx, key, b.String(), rs.ttl).Err()
}

func (rs *RedisStore[T]) Get(ctx context.Context, key string) (v *T, err error) {
	b, err := rs.c.Get(ctx, key).Bytes()
	switch err {
	case nil:
		break
	case redis.Nil:
		return nil, ErrSessionNotFound
	default:
		return nil, err
	}
	v = new(T)
	return v, gob.NewDecoder(bytes.NewBuffer(b)).Decode(v)
}

func (rs *RedisStore[T]) Del(ctx context.Context, key string) error {
	err := rs.c.Del(ctx, key).Err()
	switch err {
	case redis.Nil:
		return ErrSessionNotFound
	default:
		return err
	}
}

func (rs *RedisStore[T]) SetTTL(ttl time.Duration) { rs.ttl = ttl }

type MemStore[T any] struct {
	mu  sync.RWMutex
	m   map[string]*memstoreValue[T]
	ttl time.Duration
}

type memstoreValue[T any] struct {
	v  *T
	ts time.Time
}

func (ms *MemStore[T]) Set(ctx context.Context, key string, val *T) error {
	v := &memstoreValue[T]{v: val, ts: time.Now()}
	ms.mu.Lock()
	ms.m[key] = v
	ms.mu.Unlock()
	return nil
}

func (ms *MemStore[T]) Get(ctx context.Context, key string) (*T, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	v, ok := ms.m[key]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return v.v, nil
}

func (ms *MemStore[T]) Del(ctx context.Context, key string) error {
	ms.mu.Lock()
	delete(ms.m, key)
	ms.mu.Unlock()
	return nil
}

func (ms *MemStore[T]) SetTTL(ttl time.Duration) { ms.ttl = ttl }

func (ms *MemStore[T]) tidy(period time.Duration) {
	ticker := time.NewTicker(period)
	for {
		<-ticker.C
		if ms.ttl == -1 {
			continue
		}
		now := time.Now()
		for key, val := range ms.m {
			if now.After(val.ts.Add(ms.ttl)) {
				ms.mu.Lock()
				delete(ms.m, key)
				ms.mu.Unlock()
			}
		}
	}
}
