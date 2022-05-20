package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

type Manager[T any] struct {
	Store Store[T]
	GenID func() string
	Name  string
	opts  *CookieOptions
}

func (m *Manager[T]) NewSession(v *T) *Session[T] {
	if v == nil {
		v = new(T)
	}
	return m.newSession(m.GenID(), v)
}

func (m *Manager[T]) Get(r *http.Request) (*Session[T], error) {
	c, err := r.Cookie(m.Name)
	if err != nil {
		return nil, err
	}
	val, err := m.Store.Get(r.Context(), c.Value)
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
	if err = m.Store.Del(r.Context(), c.Value); err != nil {
		return err
	}
	c.Expires = time.Unix(0, 0)
	c.Value = ""
	http.SetCookie(w, c)
	return nil
}

func (m *Manager[T]) NewValue(w http.ResponseWriter, r *http.Request, value *T) error {
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
	return m.Store.Get(r.Context(), c.Value)
}

func (m *Manager[T]) newSession(id string, val *T) *Session[T] {
	return &Session[T]{
		Value: val,
		Opts:  *m.opts,
		name:  m.Name,
		id:    id,
		store: m.Store,
	}
}

func (m *Manager[T]) set(ctx context.Context, w http.ResponseWriter, id string, value *T) error {
	err := m.Store.Set(ctx, id, value)
	if err != nil {
		return err
	}
	http.SetCookie(w, m.opts.newCookie(m.Name, id))
	return nil
}

type CookieOptions struct {
	Path       string
	Domain     string
	Expiration time.Duration
	MaxAge     int
	HTTPOnly   bool
	SameSite   http.SameSite
	Secure     bool
}

func (co *CookieOptions) newCookie(name, value string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     co.Path,
		Domain:   co.Domain,
		Expires:  time.Now().Add(co.Expiration),
		MaxAge:   co.MaxAge,
		HttpOnly: co.HTTPOnly,
		SameSite: co.SameSite,
		Secure:   co.Secure,
	}
}

func defaultIDGenerator() string {
	var b [32]byte
	rand.Read(b[:])
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

func (s *Session[T]) Save(ctx context.Context, w http.ResponseWriter) error {
	err := s.store.Set(ctx, s.id, s.Value)
	if err != nil {
		return err
	}
	http.SetCookie(w, s.Opts.newCookie(s.name, s.id))
	return nil
}

func (s *Session[T]) Delete(ctx context.Context, w http.ResponseWriter) error {
	err := s.store.Del(ctx, s.id)
	if err != nil {
		return err
	}
	http.SetCookie(w, s.Opts.newCookie(s.name, ""))
	return nil
}
