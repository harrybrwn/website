package app

import (
	"os"

	"github.com/sirupsen/logrus"
	"harrybrown.com/app/chat"
	"harrybrown.com/pkg/log"
)

var (
	Domain = "localhost:8080"
	logger = log.GetLogger()
)

func init() {
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
