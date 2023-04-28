package session

import (
	"net/http"
	"time"
)

type CookieOpt func(*CookieOptions)

func WithPath(p string) CookieOpt              { return func(co *CookieOptions) { co.Path = p } }
func WithDomain(d string) CookieOpt            { return func(co *CookieOptions) { co.Domain = d } }
func WithExpiration(e time.Duration) CookieOpt { return func(co *CookieOptions) { co.Expiration = e } }
func WithMaxAge(v int) CookieOpt               { return func(co *CookieOptions) { co.MaxAge = v } }
func WithHTTPOnly(v bool) CookieOpt            { return func(co *CookieOptions) { co.HTTPOnly = v } }
func WithSameSite(v http.SameSite) CookieOpt   { return func(co *CookieOptions) { co.SameSite = v } }
func WithSecure(v bool) CookieOpt              { return func(co *CookieOptions) { co.Secure = v } }

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
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     co.Path,
		Domain:   co.Domain,
		MaxAge:   co.MaxAge,
		HttpOnly: co.HTTPOnly,
		SameSite: co.SameSite,
		Secure:   co.Secure,
	}
	if co.Expiration != 0 {
		c.Expires = time.Now().Add(co.Expiration)
	}
	return c
}

func unsetCookie(w http.ResponseWriter, c *http.Cookie) {
	c.Expires = time.Unix(0, 0)
	c.Value = ""
	http.SetCookie(w, c)
}
