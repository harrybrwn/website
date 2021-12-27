package log

import (
	"io"

	"github.com/labstack/echo/v4"
	echolog "github.com/labstack/gommon/log"
	"github.com/sirupsen/logrus"
)

func WrapLogrus(logger *logrus.Logger) echo.Logger {
	return &logrusLogger{Logger: logger}
}

type logrusLogger struct {
	*logrus.Logger
}

func (l *logrusLogger) Level() echolog.Lvl {
	return echolog.Lvl(l.Logger.Level)
}

func (l *logrusLogger) SetLevel(lvl echolog.Lvl) {
	l.Logger.SetLevel(echoLvlToLogrusLevel(lvl))
}

func (l *logrusLogger) Output() io.Writer {
	return l.Logger.Out
}

func (l *logrusLogger) Prefix() string   { return "" }
func (l *logrusLogger) SetPrefix(string) {}
func (l *logrusLogger) SetHeader(string) {}

func (l *logrusLogger) Printj(j echolog.JSON) { l.WithFields(logrus.Fields(j)).Print("") }
func (l *logrusLogger) Debugj(j echolog.JSON) { l.WithFields(logrus.Fields(j)).Debug("") }
func (l *logrusLogger) Infoj(j echolog.JSON)  { l.WithFields(logrus.Fields(j)).Info("") }
func (l *logrusLogger) Warnj(j echolog.JSON)  { l.WithFields(logrus.Fields(j)).Warn("") }
func (l *logrusLogger) Errorj(j echolog.JSON) { l.WithFields(logrus.Fields(j)).Error("") }
func (l *logrusLogger) Fatalj(j echolog.JSON) { l.WithFields(logrus.Fields(j)).Fatal("") }
func (l *logrusLogger) Panicj(j echolog.JSON) { l.WithFields(logrus.Fields(j)).Panic("") }

func echoLvlToLogrusLevel(lvl echolog.Lvl) logrus.Level {
	switch lvl {
	case echolog.DEBUG:
		return logrus.DebugLevel
	case echolog.INFO:
		return logrus.InfoLevel
	case echolog.WARN:
		return logrus.WarnLevel
	case echolog.ERROR:
		return logrus.ErrorLevel
	case echolog.OFF:
		return logrus.FatalLevel
	default:
		return logrus.TraceLevel
	}
}
