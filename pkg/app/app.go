package app

import (
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"gopkg.hrry.dev/homelab/pkg/app/chat"
	"gopkg.hrry.dev/homelab/pkg/log"
)

var (
	Domain = "localhost:8080"
	logger = log.GetLogger()
)

// Debug cooresponds with the debug flag
var Debug = false

func init() {
	BoolFlag(&Debug, "debug", "turn on debugging options")
	env, ok := os.LookupEnv("ENV")
	if !ok {
		return
	}
	switch env {
	case "stg", "staging":
		Domain = "staging.harrybrwn.com"
	case "prd", "prod", "production":
		Domain = "harrybrwn.com"
	case "local", "dev", "development":
		fallthrough
	default:
		Domain = "localhost:8080"
	}
}

func SetLogger(l *logrus.Logger) {
	logger = l
	chat.SetLogger(l)
}

type Pingable interface {
	Ping() error
}

func Ready(db Pingable, rd redis.Cmdable) echo.HandlerFunc {
	return func(c echo.Context) error {
		blob := `{"status":"ok"}`
		status := 200
		err := rd.Ping(c.Request().Context()).Err()
		if err != nil {
			blob = `{"status":"waiting","reason":"redis"}`
			status = http.StatusServiceUnavailable
		}
		err = db.Ping()
		if err != nil {
			blob = `{"status":"waiting","reason":"postgres"}`
			status = http.StatusServiceUnavailable
		}
		return c.Blob(status, "application/json", []byte(blob))
	}
}

func Alive(c echo.Context) error {
	return c.Blob(200, "application/json", []byte(`{"status":"ok"}`))
}
