package app

import (
	"math"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/db"
)

const (
	tokenKey     = "_token"
	maxCookieAge = 2147483647
)

var logger = logrus.New()

func SetLogger(l *logrus.Logger) { logger = l }

func TokenHandler(conf auth.TokenConfig, store UserStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			err         error
			body        Login
			req         = c.Request()
			ctx         = req.Context()
			cookieQuery = req.URL.Query().Get("cookie")
			setCookie   bool
		)
		switch err = c.Bind(&body); err {
		case nil:
			break
		case echo.ErrUnsupportedMediaType:
			return err
		default:
			err = errors.Wrap(err, "failed to bind user data")
			return wrap(err, http.StatusInternalServerError)
		}
		logger := logger.WithFields(logrus.Fields{
			"username": body.Username,
			"email":    body.Email,
		})
		if len(cookieQuery) > 0 {
			setCookie, err = strconv.ParseBool(cookieQuery)
			if err != nil {
				return echo.ErrBadRequest.SetInternal(err)
			}
		} else {
			setCookie = false
		}
		logger.Info("getting token")
		u, err := store.Login(ctx, &body)
		if err != nil {
			return echo.ErrNotFound.SetInternal(errors.Wrap(err, "failed to login"))
		}
		claims := u.NewClaims()
		resp, err := auth.NewTokenResponse(conf, claims)
		if err != nil {
			return echo.ErrInternalServerError.SetInternal(
				errors.Wrap(err, "could not create token response"))
		}
		c.Set(auth.ClaimsContextKey, claims)
		if setCookie {
			logger.Info("setting cookie")
			c.SetCookie(&http.Cookie{
				Name:    tokenKey,
				Value:   resp.Token,
				Expires: claims.ExpiresAt.Time,
				Path:    "/",
			})
		} else {
			logger.Info("not sending cookie")
		}
		return c.JSON(200, resp)
	}
}

const hitsQuery = `select count(*) from request_log where uri = $1`

func Hits(d db.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			n   int
			u   = c.QueryParam("u")
			ctx = c.Request().Context()
		)
		if len(u) == 0 {
			u = "/"
		}
		rows, err := d.QueryContext(ctx, hitsQuery, u)
		if err != nil {
			return wrap(err, 500, "could not execute query hits")
		}
		if err = db.ScanOne(rows, &n); err != nil {
			return wrap(err, 500, "could not scan row")
		}
		return c.JSON(200, map[string]int{"count": n})
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

func HandleInfo(w http.ResponseWriter, r *http.Request) interface{} {
	return Info{
		Name: "Harry Brown",
		Age:  math.Round(GetAge()),
	}
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
	4, 40, 0, 0, // 4:40 AM
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
		// {Body: "I did not have sexual relations with that woman.", Author: "Bill Clinton"},
		// {Body: "Bush did 911.", Author: "A very intelligent internet user"},
	}
)

func RandomQuote() Quote {
	quotesMu.Lock()
	defer quotesMu.Unlock()
	return quotes[rand.Intn(len(quotes))]
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
