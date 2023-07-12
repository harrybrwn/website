package app

import (
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"gopkg.hrry.dev/homelab/pkg/session"
	"gopkg.hrry.dev/homelab/pkg/web"
)

type SessionData struct {
	// User *User
	Hits int
}

type SessionManager = session.Manager[SessionData]

func NewSessionManager(rd redis.UniversalClient, cookieDomain string) *SessionManager {
	return session.NewManager(
		"session",
		session.NewStore[SessionData](rd, time.Hour*24*365),
		session.WithDomain(cookieDomain),
		session.WithPath("/"),
		session.WithSameSite(http.SameSiteNoneMode),
		session.WithSecure(true),
	)
}

func Session(m *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ss, err := m.Get(r)
		if err == session.ErrSessionNotFound || err == http.ErrNoCookie {
			ss = m.NewSession(&SessionData{})
		} else if err != nil {
			web.WriteError(w, err)
			return
		}
		err = ss.Save(r.Context(), w)
		if err != nil {
			web.WriteError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func CollectSession(m *SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ss, err := m.Get(r)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			ctx := session.StashInContext(r.Context(), ss)
			next.ServeHTTP(w, r.WithContext(ctx))
			ss.Value.Hits++
			err = ss.Save(r.Context(), w)
			if err != nil {
				logger.WithError(err).Error("failed to save session data")
				return
			}
		})
	}
}
