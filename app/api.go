package app

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	mrand "math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
	"harrybrown.com/app/chat"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/db"
	"nhooyr.io/websocket"
)

const (
	tokenKey = "_token"
	//maxCookieAge = 2147483647
)

func NewTokenConfig() auth.TokenConfig {
	hexseed, hasSeed := os.LookupEnv("JWT_SEED")
	if hasSeed {
		logger.Info("creating token config from seed")
		seed, err := hex.DecodeString(hexseed)
		if err != nil {
			panic(errors.Wrap(err, "could not decode private key seed from hex"))
		}
		return &tokenConfig{auth.EdDSATokenConfigFromSeed(seed)}
	}
	logger.Warn("generating new key pair for token config")
	return &tokenConfig{auth.GenEdDSATokenConfig()}
}

type tokenConfig struct{ auth.TokenConfig }

func (tc *tokenConfig) GetToken(r *http.Request) (string, error) {
	c, err := r.Cookie(tokenKey)
	if err != nil {
		return auth.GetBearerToken(r)
	}
	return c.Value, nil
}

type TokenService struct {
	Tokens auth.TokenStore
	Users  UserStore
	Config auth.TokenConfig
}

func (ts *TokenService) Token(c echo.Context) error {
	var (
		err  error
		body Login
		req  = c.Request()
		ctx  = req.Context()
	)
	switch err = c.Bind(&body); err {
	case nil:
		break
	case echo.ErrUnsupportedMediaType:
		return err
	default:
		err = errors.Wrap(err, "failed to bind user data")
		return echo.ErrInternalServerError.SetInternal(err)
	}
	logger := logger.WithFields(logrus.Fields{
		"username": body.Username,
		"email":    body.Email,
	})
	setCookie, err := ts.parserCookieQuery(req)
	if err != nil {
		return err
	}
	logger.Info("getting token")
	u, err := ts.Users.Login(ctx, &body)
	if err != nil {
		return echo.ErrNotFound.SetInternal(errors.Wrap(err, "failed to login"))
	}

	claims := u.NewClaims()
	// Generate both a new access token and refresh token.
	resp, err := auth.NewTokenResponse(ts.Config, claims)
	if err != nil {
		return echo.ErrInternalServerError.SetInternal(
			errors.Wrap(err, "could not create token response"))
	}
	err = ts.Tokens.Set(ctx, u.ID, resp.RefreshToken)
	if err != nil {
		return echo.ErrInternalServerError.SetInternal(err)
	}
	c.Set(auth.ClaimsContextKey, claims)
	if setCookie {
		ts.setTokenCookie(c.Response(), resp, claims)
	}
	return c.JSON(200, resp)
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (ts *TokenService) Refresh(c echo.Context) error {
	var (
		err      error
		req      = c.Request()
		ctx      = req.Context()
		tokenReq RefreshTokenReq
	)
	err = c.Bind(&tokenReq)
	if err != nil {
		return echo.ErrBadRequest.SetInternal(err)
	}
	setCookie, err := ts.parserCookieQuery(req)
	if err != nil {
		return err
	}

	logger.WithField("refresh_token", tokenReq.RefreshToken).Info("received refresh token")
	refreshClaims, err := auth.ValidateRefreshToken(tokenReq.RefreshToken, ts.keyfunc)
	if err != nil {
		return echo.ErrBadRequest.SetInternal(err)
	}

	stored, err := ts.Tokens.Get(ctx, refreshClaims.ID)
	if err != nil {
		return echo.ErrUnauthorized.SetInternal(err)
	}
	if tokenReq.RefreshToken != stored {
		logger.Warn("refresh token did not match up with token in storage")
		return echo.ErrUnauthorized
	}

	now := time.Now()
	claims := auth.Claims{
		ID:    refreshClaims.ID,
		UUID:  refreshClaims.UUID,
		Roles: refreshClaims.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  []string{auth.TokenAudience},
			Issuer:    auth.Issuer,
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	// Only generate a new access token, the client should still have the refresh token.
	resp, err := auth.NewAccessToken(ts.Config, &claims)
	if err != nil {
		return echo.ErrInternalServerError.SetInternal(err)
	}
	// Send back the same refresh token they sent
	resp.RefreshToken = tokenReq.RefreshToken

	c.Set(auth.ClaimsContextKey, claims)
	if setCookie {
		ts.setTokenCookie(c.Response(), resp, &claims)
	}
	return c.JSON(200, resp)
}

func (ts *TokenService) parserCookieQuery(req *http.Request) (bool, error) {
	var (
		set         bool
		err         error
		cookieQuery = req.URL.Query().Get("cookie")
	)
	if len(cookieQuery) > 0 {
		set, err = strconv.ParseBool(cookieQuery)
		if err != nil {
			return false, echo.ErrBadRequest.SetInternal(err)
		}
	} else {
		set = false
	}
	return set, nil
}

func (ts *TokenService) setTokenCookie(response http.ResponseWriter, token *auth.TokenResponse, claims *auth.Claims) {
	http.SetCookie(response, &http.Cookie{
		Name:     tokenKey,
		Value:    token.Token,
		Expires:  claims.ExpiresAt.Time,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})
}

func (ts *TokenService) Revoke(c echo.Context) error {
	var (
		ctx = c.Request().Context()
		req RefreshTokenReq
	)
	err := c.Bind(&req)
	if err != nil {
		return err
	}
	claims := auth.GetClaims(c)
	if claims == nil {
		return echo.ErrBadRequest
	}
	token, err := ts.Tokens.Get(ctx, claims.ID)
	if err != nil {
		return err
	}
	if req.RefreshToken != token {
		return &echo.HTTPError{
			Code:    http.StatusBadRequest,
			Message: "incorrect refresh token",
		}
	}
	err = ts.Tokens.Del(c.Request().Context(), claims.ID)
	if err != nil {
		return echo.ErrInternalServerError
	}
	return nil
}

func (ts *TokenService) keyfunc(*jwt.Token) (interface{}, error) {
	return ts.Config.Public(), nil
}

type EmailClient interface {
	SendWithContext(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error)
}

func SendMail(client EmailClient) echo.HandlerFunc {
	type addr struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	}
	type Body struct {
		From    addr   `json:"from"`
		To      addr   `json:"to"`
		Subject string `json:"subject"`
		Content string `json:"content"`
	}
	return func(c echo.Context) error {
		var (
			err error
			b   Body
			ctx = c.Request().Context()
		)
		if err = c.Bind(&b); err != nil {
			return err
		}
		from := mail.NewEmail(b.From.Name, b.From.Address)
		to := mail.NewEmail(b.To.Name, b.To.Address)
		message := mail.NewSingleEmail(from, b.Subject, to, "", b.Content)
		response, err := client.SendWithContext(ctx, message)
		if err != nil {
			return err
		}
		return c.JSON(200, response)
	}
}

func CreateChatRoom(store chat.Store) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := auth.GetClaims(c)
		if claims == nil {
			return echo.ErrUnauthorized
		}
		name := c.QueryParam("name")
		if name == "" {
			return echo.ErrBadRequest
		}
		room, err := store.CreateRoom(c.Request().Context(), claims.ID, name)
		if err != nil {
			return echo.ErrInternalServerError
		}
		return c.JSON(200, room)
	}
}

const hitsQuery = `SELECT count(*) FROM request_log WHERE uri = $1`

func NewHitsCache(rd redis.Cmdable) HitsCache {
	return &hitsCache{rd: rd, timeout: time.Hour}
}

type HitsCache interface {
	Next(context.Context, string) (int64, error)
	Put(context.Context, string, int64) error
}

type hitsCache struct {
	rd      redis.Cmdable
	timeout time.Duration
}

func (hc *hitsCache) Next(ctx context.Context, k string) (int64, error) {
	count, err := hc.rd.Incr(ctx, k).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		return 0, errors.New("increment not yet set")
	}
	return count, nil
}

func (hc *hitsCache) Put(ctx context.Context, k string, n int64) error {
	return hc.rd.Set(ctx, k, n, hc.timeout).Err()
}

func Hits(d db.DB, h HitsCache, logger logrus.FieldLogger) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			n   int64
			u   = c.QueryParam("u")
			ctx = c.Request().Context()
		)
		if len(u) == 0 {
			u = "/"
		}
		key := fmt.Sprintf("hits:%s", u)
		count, err := h.Next(ctx, key)
		if err == nil {
			return c.JSON(200, map[string]int64{"count": count})
		}
		rows, err := d.QueryContext(ctx, hitsQuery, u)
		if err != nil {
			return wrap(err, 500, "could not execute query hits")
		}
		if err = db.ScanOne(rows, &n); err != nil {
			return wrap(err, 500, "could not scan row")
		}
		err = h.Put(ctx, key, n)
		if err != nil {
			logger.WithError(err).Warn("could not set cached page views")
		}
		return c.JSON(200, map[string]int64{"count": n})
	}
}

func LogListHandler(db db.DB) echo.HandlerFunc {
	logs := LogManager{db: db, logger: logger}
	type listquery struct {
		Limit  int  `query:"limit"`
		Offset int  `query:"offset"`
		Rev    bool `query:"rev"`
	}
	return func(c echo.Context) error {
		var q listquery
		err := c.Bind(&q)
		if err != nil {
			return err
		}
		if q.Limit == 0 {
			q.Limit = 20
		}
		logs, err := logs.Get(c.Request().Context(), q.Limit, q.Offset, q.Rev)
		if err != nil {
			return err
		}
		return c.JSON(200, logs)
	}
}

func ChatSocketHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ws, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{})
		if err != nil {
			return echo.ErrInternalServerError.SetInternal(err)
		}
		return ws.Close(websocket.StatusNormalClosure, "not implemented yet")
	}
}

func HandleInfo(w http.ResponseWriter, r *http.Request) interface{} {
	return Info{
		Name: "Harry Brown",
		Age:  math.Round(GetAge()),
	}
}

func HandleRuntimeInfo(startup time.Time) echo.HandlerFunc {
	return func(c echo.Context) error { return c.JSON(200, RuntimeInfo(startup)) }
}

func RuntimeInfo(start time.Time) *Info {
	return &Info{
		Name:      "Harry Brown",
		Age:       GetAge(),
		Birthday:  GetBirthday(),
		GOVersion: runtime.Version(),
		Uptime:    time.Since(start),
		Debug:     Debug,
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}
}

type Info struct {
	Name      string        `json:"name,omitempty"`
	Age       float64       `json:"age,omitempty"`
	Uptime    time.Duration `json:"uptime,omitempty"`
	GOVersion string        `json:"goversion,omitempty"`
	Error     string        `json:"error,omitempty"`
	Birthday  time.Time     `json:"birthday,omitempty"`
	Debug     bool          `json:"debug"`
	GOOS      string        `json:"GOOS,omitempty"`
	GOARCH    string        `json:"GOARCH,omitempty"`
}

var birthTimestamp = time.Date(
	1998, time.August, 4, // 1998-08-04
	4, 40, 3, 0, // 4:40:03 AM
	mustLoadLocation("America/Los_Angeles"),
)

func mustLoadLocation(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}

func GetAge() float64 {
	return time.Since(birthTimestamp).Seconds() / 60 / 60 / 24 / 365
}

func GetBirthday() time.Time { return birthTimestamp }

type Quote struct {
	Body   string `json:"body"`
	Author string `json:"author"`
}

var (
	quotesMu sync.Mutex
	quotes   = []Quote{
		{Body: "Do More", Author: "Casey Neistat"},
		{Body: "Imagination is something you do alone.", Author: "Steve Wazniak"},
		{Body: "I was never really good at anything except for the ability to learn.", Author: "Kanye West"},
		{Body: "I love sleep; It's my favorite.", Author: "Kanye West"},
		{Body: "I'm gunna be the next hokage!", Author: "Naruto Uzumaki"},
		{
			Body: "I am so clever that sometimes I don't understand a single word of " +
				"what I am saying.",
			Author: "Oscar Wilde",
		},
		{
			Body: "Have you ever had a dream that, that, um, that you had, uh, " +
				"that you had to, you could, you do, you wit, you wa, you could " +
				"do so, you do you could, you want, you wanted him to do you so much " +
				"you could do anything?",
			Author: "That one kid",
		},
		// {Body: "640K ought to be enough memory for anybody.", Author: "Bill Gates"},
		// {Body: "I did not have sexual relations with that woman.", Author: "Bill Clinton"},
		// {Body: "Bush did 911.", Author: "A very intelligent internet user"},
	}
)

func RandomQuote() Quote {
	quotesMu.Lock()
	defer quotesMu.Unlock()
	return quotes[mrand.Intn(len(quotes))]
}

func GetQuotes() []Quote {
	return quotes
}

func wrap(err error, status int, message ...string) error {
	var msg string
	if len(message) < 1 {
		msg = http.StatusText(status)
	} else {
		msg = message[0]
	}
	if err == nil {
		err = errors.New(msg)
		msg = ""
	}
	return &echo.HTTPError{
		Code:     status,
		Message:  msg,
		Internal: err,
	}
}
