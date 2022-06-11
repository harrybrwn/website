package log

import (
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/syslog"
	"github.com/sirupsen/logrus/hooks/writer"
)

func TestCopyLogger(t *testing.T) {
	is := is.New(t)
	l := logrus.New()
	var (
		exit   = func(n int) {}
		format = &logrus.JSONFormatter{TimestampFormat: time.RFC822Z}
	)
	l.ExitFunc = exit
	l.Formatter = format
	l.AddHook(&writer.Hook{})
	l.AddHook(&syslog.SyslogHook{})

	lg := logrusCopy(l)
	is.True(l != lg)
	is.Equal(l.Out, lg.Out)
	is.Equal(l.Level, lg.Level)
	is.Equal(l.Formatter, lg.Formatter)
	is.Equal(l.ExitFunc, lg.ExitFunc)
	is.Equal(l.ReportCaller, lg.ReportCaller)
	is.Equal(len(l.Hooks), len(lg.Hooks))
	for level, hooks := range l.Hooks {
		is.Equal(len(hooks), len(lg.Hooks[level]))
		for i, h := range hooks {
			is.Equal(h, lg.Hooks[level][i])
		}
	}
}
