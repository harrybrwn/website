package app

import (
	"context"
	"fmt"
	"math"
	mrand "math/rand"
	"net/http"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"

	"gopkg.hrry.dev/homelab/pkg/db"
)

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
	info := &Info{
		Name:      "Harry Brown",
		Age:       GetAge(),
		Birthday:  GetBirthday(),
		GOVersion: runtime.Version(),
		Uptime:    time.Since(start),
		Debug:     Debug,
	}
	buildinfo, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}
	info.Build = make(map[string]interface{})
	for _, setting := range buildinfo.Settings {
		info.Build[setting.Key] = setting.Value
	}
	info.Dependencies = buildinfo.Deps
	info.Module = buildinfo.Main
	info.GOVersion = buildinfo.GoVersion
	return info
}

type Info struct {
	Name         string                 `json:"name,omitempty"`
	Age          float64                `json:"age,omitempty"`
	Uptime       time.Duration          `json:"uptime,omitempty"`
	GOVersion    string                 `json:"goversion,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Birthday     time.Time              `json:"birthday,omitempty"`
	Debug        bool                   `json:"debug"`
	Build        map[string]interface{} `json:"build,omitempty"`
	Dependencies []*debug.Module        `json:"dependencies,omitempty"`
	Module       debug.Module           `json:"module,omitempty"`
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
		{Body: "640K ought to be enough memory for anybody.", Author: "Bill Gates"},
		// {Body: "I did not have sexual relations with that woman.", Author: "Bill Clinton"},
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
