package main

import (
	"errors"
	"sync"
	"time"
)

type session struct {
	state   string
	user    string
	code    string
	expires time.Time
}

type sessionStore struct {
	mu sync.RWMutex
	m  map[string]*session
}

func newSessionStore() *sessionStore {
	store := sessionStore{m: make(map[string]*session)}
	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				for k, s := range store.m {
					if now.After(s.expires) {
						store.mu.Lock()
						delete(store.m, k)
						store.mu.Unlock()
					}
				}
			}
		}
	}()
	return &store
}

func (ss *sessionStore) create(s *session) {
	ss.mu.Lock()
	s.expires = time.Now().Add(time.Minute * 5)
	ss.m[s.state] = s
	ss.mu.Unlock()
}

func (ss *sessionStore) get(key string) (*session, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	s, ok := ss.m[key]
	if !ok {
		return nil, errors.New("oauth session not found")
	}
	return s, nil
}

func (ss *sessionStore) del(s *session) {
	ss.mu.Lock()
	delete(ss.m, s.state)
	ss.mu.Unlock()
}

func (ss *sessionStore) setCode(key, code string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	s, ok := ss.m[key]
	if !ok {
		return errors.New("oauth session not found")
	}
	s.code = code
	return nil
}
