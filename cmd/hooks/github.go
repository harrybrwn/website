package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/go-github/v43/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"harrybrown.com/pkg/session"
)

type GithubAuthService struct {
	Sessions    session.Store[ghSession]
	AuthSession session.Store[oauthSession]
	Config      oauth2.Config
}

type oauthSession struct {
	state   string
	user    string
	code    string
	expires time.Time
}

type ghSession struct {
	token    oauth2.Token
	username string
}

const ghSessionKey = "gh_s"

func GithubLoggedIn(r *http.Request) bool {
	cookie, err := r.Cookie(ghSessionKey)
	if err != nil {
		return false
	}
	if cookie.Expires.Before(time.Now()) {
		return false
	}
	return true
}

func (gs *GithubAuthService) client(ctx context.Context, tok *oauth2.Token) *github.Client {
	return github.NewClient(gs.Config.Client(ctx, tok))
}

func (gs *GithubAuthService) session(ctx context.Context, r *http.Request) (*ghSession, error) {
	cookie, err := r.Cookie(ghSessionKey)
	if err != nil {
		return nil, err
	}
	s, err := gs.Sessions.Get(ctx, cookie.Value)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (gs *GithubAuthService) token(ctx context.Context, r *http.Request) (*oauth2.Token, error) {
	s, err := gs.session(ctx, r)
	if err != nil {
		return nil, err
	}
	return &s.token, nil
}

func (gs *GithubAuthService) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		SendError(w, http.StatusBadRequest, err)
		return
	}
	username := r.FormValue("username")
	session := oauthSession{state: getState()}
	err := gs.AuthSession.Set(r.Context(), session.state, &session)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err)
		return
	}
	opts := []oauth2.AuthCodeOption{allowSignUp(false)}
	if len(username) > 0 {
		opts = append(opts, oauth2.SetAuthURLParam("login", username))
	}
	loc := gs.Config.AuthCodeURL(session.state, opts...)
	redirect(w, loc)
}

func (gs *GithubAuthService) Authorize(w http.ResponseWriter, r *http.Request) {
	var err error
	if err = r.ParseForm(); err != nil {
		SendError(w, http.StatusInternalServerError, err, "could not parse form")
		return
	}
	code := r.FormValue("code")
	state := r.FormValue("state")
	if len(code) == 0 || len(state) == 0 {
		logger.WithFields(logrus.Fields{
			"error":       r.URL.Query().Get("error"),
			"description": r.URL.Query().Get("error_description"),
		})
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	ctx := r.Context()
	authSess, err := gs.AuthSession.Get(ctx, state)
	if err != nil {
		SendError(w, http.StatusNotFound, err, "could not find oauth session")
		return
	}
	err = gs.AuthSession.Del(ctx, authSess.state)
	if err != nil {
		logger.WithError(err).Warn("failed to delete oauth session")
	}
	token, err := gs.Config.Exchange(ctx, code)
	if err != nil {
		SendError(w, http.StatusUnauthorized, err)
		return
	}

	client := github.NewClient(gs.Config.Client(ctx, token))
	user, response, err := client.Users.Get(ctx, "")
	if err != nil {
		logger.WithError(err).Warn("could not find github user")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if user.Login == nil {
		SendError(w, http.StatusInternalServerError, nil)
		return
	}
	logger.WithFields(logrus.Fields{
		"oauth-scopes":          response.Header.Get("X-OAuth-Scopes"),
		"accepted-oauth-scopes": response.Header.Get("X-Accepted-OAuth-Scopes"),
		"login":                 *user.Login,
	}).Info("got user")
	id := getState()
	ghs := ghSession{token: *token, username: *user.Login}
	if err = gs.Sessions.Set(ctx, id, &ghs); err != nil {
		SendError(w, http.StatusInternalServerError, err, "failed to store new session")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     ghSessionKey,
		Value:    id,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
	redirect(w, "/")
}

func (gs *GithubAuthService) SignOut(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(ghSessionKey)
	if err != nil {
		SendError(w, http.StatusNotFound, err, "session cookie not found")
		return
	}
	// remove cookie
	cookie.Expires = time.Unix(0, 0)
	http.SetCookie(w, cookie)
	err = gs.Sessions.Del(r.Context(), cookie.Value)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func allowSignUp(val bool) oauth2.AuthCodeOption {
	return oauth2.SetAuthURLParam("allow_signup", strconv.FormatBool(val))
}

func createHook(store session.Store[ghSession], host string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			SendError(w, http.StatusUnauthorized, nil, "session cookie not found")
			return
		}
		ctx := r.Context()
		session, err := store.Get(ctx, cookie.Value)
		if err != nil {
			SendError(w, http.StatusUnauthorized, nil, "github session not found")
			return
		}
		repo := r.URL.Query().Get("repo")
		if len(repo) == 0 {
			SendError(w, http.StatusBadRequest, nil, "no repo query found")
			return
		}
		callback := r.URL.Query().Get("callback")
		if len(callback) == 0 {
			SendError(w, http.StatusBadRequest, nil, "no callback query")
			return
		}
		if callback[0] != '/' {
			SendError(w, http.StatusBadRequest, nil, "callback should start with a forward slash ('/')")
			return
		}

		client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&session.token)))
		active := true
		hook, _, err := client.Repositories.CreateHook(ctx, session.username, repo, &github.Hook{
			Config: map[string]any{
				"url": (&url.URL{
					Scheme: "https",
					Host:   host,
					Path:   callback,
				}).String(),
				"insecure_ssl": 0,
			},
			Events: []string{
				"push",
				"pull_request",
				"ping",
			},
			Active: &active,
		})
		if err != nil {
			logger.WithError(err).Error("failed to list create a webhook")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Println(hook)
	}
}
