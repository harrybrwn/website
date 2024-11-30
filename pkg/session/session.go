package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
)

func NewManager[T any](name string, store Store[T], opts ...CookieOpt) *Manager[T] {
	m := Manager[T]{
		Name:  name,
		Store: store,
		GenID: defaultIDGenerator,
		opts: &CookieOptions{
			Path:     "/",
			SameSite: http.SameSiteDefaultMode,
		},
	}
	for _, o := range opts {
		o(m.opts)
	}
	return &m
}

type Manager[T any] struct {
	Store Store[T]
	GenID func() string
	Name  string
	opts  *CookieOptions
}

func (m *Manager[T]) NewSession(v *T, opts ...CookieOpt) *Session[T] {
	if v == nil {
		v = new(T)
	}
	id := m.GenID()
	return m.newSession(id, v, opts...)
}

func (m *Manager[T]) Get(r *http.Request) (*Session[T], error) {
	c, err := r.Cookie(m.Name)
	if err != nil {
		return nil, err
	}
	val, err := m.Store.Get(r.Context(), m.key(c.Value))
	if err != nil {
		return nil, err
	}
	return m.newSession(c.Value, val), nil
}

func (m *Manager[T]) Delete(w http.ResponseWriter, r *http.Request) error {
	c, err := r.Cookie(m.Name)
	if err != nil {
		return err
	}
	if err = m.Store.Del(r.Context(), m.key(c.Value)); err != nil {
		return err
	}
	unsetCookie(w, c)
	return nil
}

func (m *Manager[T]) SetValue(w http.ResponseWriter, r *http.Request, value *T) error {
	id := m.GenID()
	return m.set(r.Context(), w, id, value)
}

func (m *Manager[T]) UpdateValue(w http.ResponseWriter, r *http.Request, value *T) error {
	c, err := r.Cookie(m.Name)
	if err != nil {
		return err
	}
	return m.set(r.Context(), w, c.Value, value)
}

func (m *Manager[T]) GetValue(r *http.Request) (*T, error) {
	c, err := r.Cookie(m.Name)
	if err != nil {
		return nil, err
	}
	return m.Store.Get(r.Context(), m.key(c.Value))
}

func (m *Manager[T]) newSession(id string, val *T, opts ...CookieOpt) *Session[T] {
	s := Session[T]{
		Value: val,
		Opts:  *m.opts,
		name:  m.Name,
		id:    id,
		store: m.Store,
	}
	for _, o := range opts {
		o(&s.Opts)
	}
	return &s
}

func (m *Manager[T]) set(ctx context.Context, w http.ResponseWriter, id string, value *T) error {
	err := m.Store.Set(ctx, m.key(id), value)
	if err != nil {
		return err
	}
	http.SetCookie(w, m.opts.newCookie(m.Name, id))
	return nil
}

func (m *Manager[T]) key(v string) string {
	return fmt.Sprintf("%s:%s", m.Name, v)
}

func defaultIDGenerator() string {
	var b [32]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

type Session[T any] struct {
	Value *T
	Opts  CookieOptions
	store Store[T]
	id    string
	name  string
}

func (s *Session[T]) ID() string   { return s.id }
func (s *Session[T]) Name() string { return s.name }
func (s *Session[T]) Set(value *T) { s.Value = value }
func (s *Session[T]) key() string  { return fmt.Sprintf("%s:%s", s.name, s.id) }

func (s *Session[T]) Save(ctx context.Context, w http.ResponseWriter) error {
	err := s.store.Set(ctx, s.key(), s.Value)
	if err != nil {
		return err
	}
	http.SetCookie(w, s.Opts.newCookie(s.name, s.id))
	return nil
}

func (s *Session[T]) Delete(ctx context.Context, w http.ResponseWriter) error {
	err := s.store.Del(ctx, s.key())
	if err != nil {
		return err
	}
	http.SetCookie(w, s.Opts.newCookie(s.name, ""))
	return nil
}

type sessionContextKeyType struct{}

var sessionContextKey sessionContextKeyType

func StashInContext[T any](ctx context.Context, s *Session[T]) context.Context {
	return context.WithValue(ctx, sessionContextKey, s)
}

func FromContext[T any](ctx context.Context) *Session[T] {
	v := ctx.Value(sessionContextKey)
	if v == nil {
		return nil
	}
	return v.(*Session[T])
}
