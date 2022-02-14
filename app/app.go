package app

import (
	"github.com/sirupsen/logrus"
	"harrybrown.com/app/chat"
)

var logger = logrus.New()

func SetLogger(l *logrus.Logger) {
	logger = l
	chat.SetLogger(l)
}
