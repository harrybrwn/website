package app

import (
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func SetLogger(l *logrus.Logger) {
	logger = l
}
