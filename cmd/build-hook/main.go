package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"harrybrown.com/pkg/log"
)

var (
	//go:embed index.html
	index []byte

	logger = log.GetLogger()
)

func main() {
	store := newSessionStore()
	client := client{
		clientID:     os.Getenv("GITHUB_CLIENT_ID"),
		clientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		port:         8889,
	}
	flag.IntVar(&client.port, "port", client.port, "specify server port")
	flag.Parse()
	if err := godotenv.Load(); err != nil {
		logger.WithError(err).Warn("could not load .env")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write(index)
	})
	mux.HandleFunc("/login/github", login(&client, store))
	mux.HandleFunc("/authorize/github", authorize(&client, store))

	addr := fmt.Sprintf(":%d", client.port)
	logger.WithFields(logrus.Fields{
		"address": addr,
		"time":    time.Now(),
	}).Info("starting server")
	http.ListenAndServe(addr, logs(mux))
}

func login(c *client, store *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			logger.WithError(err).Error("could not parse form")
		}
		username := r.PostForm.Get("username")
		logger.WithField("username", username).Info("got login request")
		scopes := []string{
			"user:email",
			"write:repo_hook",
			"read:repo_hook",
		}
		session := session{
			state: getState(),
			user:  username,
		}
		store.create(&session)
		redirect(w, c.authorizeURL(username, session.state, scopes))
	}
}

func authorize(c *client, store *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		if err = r.ParseForm(); err != nil {
			logger.WithError(err).Error("could not parse form")
			w.WriteHeader(http.StatusInternalServerError)
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
		session, err := store.get(state)
		if err != nil {
			logger.WithError(err).Error("could not find oauth session for github")
			w.WriteHeader(404)
			return
		}
		store.del(session)
		token, err := c.getToken(code)
		if err != nil {
			logger.WithError(err).Error("could not fetch github access token")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Printf("%#v\n", token)
		w.WriteHeader(200)
		redirect(w, "/")
	}
}

type token struct {
	Token string `json:"access_token"`
	Scope string `json:"scope"`
	Type  string `json:"token_type"`
}

type client struct {
	clientID     string
	clientSecret string
	port         int
}

func (c *client) getToken(code string) (*token, error) {
	res, err := http.DefaultClient.Do(&http.Request{
		Method: "POST",
		Header: http.Header{
			"accept": {"application/json"},
		},
		URL: &url.URL{
			Scheme: "https",
			Host:   "github.com",
			Path:   "/login/oauth/access_token",
			RawQuery: (&url.Values{
				"client_id":     {c.clientID},
				"client_secret": {c.clientSecret},
				"code":          {code},
			}).Encode(),
		},
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var t token
	err = json.NewDecoder(res.Body).Decode(&t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (c *client) authorizeURL(user, state string, scopes []string) string {
	query := url.Values{
		"client_id":    {c.clientID},
		"redirect_uri": {fmt.Sprintf("http://127.0.0.1:%d/authorize/github", c.port)},
		"state":        {state},
		"scope":        {strings.Join(scopes, " ")},
		"login":        {user},
		"allow_signup": {"false"},
	}
	u := url.URL{
		Scheme:   "https",
		Host:     "github.com",
		Path:     "/login/oauth/authorize",
		RawQuery: query.Encode(),
	}
	fmt.Println(u.String())
	return u.String()
}

func redirect(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func logs(h http.Handler) http.HandlerFunc {
	logger := log.GetLogger()
	return func(w http.ResponseWriter, r *http.Request) {
		logger.WithFields(logrus.Fields{
			"uri":         r.RequestURI,
			"host":        r.Host,
			"status":      r.Response.StatusCode,
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"referer":     r.Referer(),
			"query":       r.URL.RawQuery,
		}).Info("request")
		h.ServeHTTP(w, r)
	}
}

func getState() string {
	var b [32]byte
	rand.Read((b[:]))
	return hex.EncodeToString(b[:])
}
