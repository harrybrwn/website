package app

import (
	"github.com/sirupsen/logrus"
	"harrybrown.com/app/chat"
	"harrybrown.com/pkg/log"
)

var logger = log.GetLogger()

func SetLogger(l *logrus.Logger) {
	logger = l
	chat.SetLogger(l)
}
