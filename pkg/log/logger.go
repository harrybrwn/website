package log

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var std = logrus.StandardLogger()

type (
	FieldLogger = logrus.FieldLogger
	Logger      = logrus.Logger
	Fields      = logrus.Fields
	Level       = logrus.Level
)

type LoggerOpt func(*Logger)

func New(opts ...LoggerOpt) *Logger {
	l := logrus.New()
	for _, o := range opts {
		o(l)
	}
	return l
}

func SetLogger(l *Logger) *Logger {
	std = l
	return std
}

func GetLogger() *Logger {
	return std
}

// WithEnv will configure the logger using environment variables.
func WithEnv() LoggerOpt {
	lvl, err := parseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		std.Fatalf("Error: %v", err)
	}
	var format Format
	switch f := strings.ToLower(os.Getenv("LOG_FORMAT")); f {
	case "json", "":
		format = JSONFormat
	case "text":
		format = TextFormat
	default:
		std.Fatalf("Error: invalid logging format %q", f)
	}
	return func(l *Logger) {
		WithLevel(lvl)(l)
		WithFormat(format)(l)
	}
}

func WithFields(fields Fields) LoggerOpt {
	return func(l *Logger) {
		l.AddHook(&fieldsHook{fields: fields})
	}
}

func WithServiceName(name string) LoggerOpt {
	return func(l *Logger) {
		l.AddHook(&fieldsHook{fields: Fields{
			"service": name,
		}})
	}
}

const (
	PanicLevel Level = logrus.PanicLevel
	FatalLevel Level = logrus.FatalLevel
	ErrorLevel Level = logrus.ErrorLevel
	WarnLevel  Level = logrus.WarnLevel
	InfoLevel  Level = logrus.InfoLevel
	DebugLevel Level = logrus.DebugLevel
	TraceLevel Level = logrus.TraceLevel
)

func WithLevel(level Level) LoggerOpt { return func(l *Logger) { l.SetLevel(level) } }

type Format int

const (
	JSONFormat Format = iota
	TextFormat
)

var (
	TextFormatter = logrus.TextFormatter{}
	JSONFormatter = logrus.JSONFormatter{TimestampFormat: time.RFC3339}
)

func WithFormat(format Format) LoggerOpt {
	return func(l *Logger) {
		switch format {
		case JSONFormat:
			l.SetFormatter(&JSONFormatter)
		case TextFormat:
			l.SetFormatter(&TextFormatter)
		}
	}
}

func parseLevel(l string) (Level, error) {
	switch strings.ToLower(l) {
	case "":
		return InfoLevel, nil
	case "panic":
		return PanicLevel, nil
	case "fatal":
		return FatalLevel, nil
	case "error":
		return ErrorLevel, nil
	case "warn":
		return WarnLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	case "trace":
		return TraceLevel, nil
	default:
		return TraceLevel, fmt.Errorf("invalid logging format %q", l)
	}
}

func GetOutput(envkey string) io.Writer {
	out, ok := os.LookupEnv(envkey)
	if !ok {
		return os.Stdout
	}
	fdout := strings.ToLower(out)
	if fdout == "1" || fdout == "stdout" {
		return os.Stdout
	} else if fdout == "2" || fdout == "stderr" {
		return os.Stderr
	}
	file, err := os.Open(out)
	if err != nil {
		std.Warnf("failed to open log file: %v", err)
		return os.Stdout
	}
	// TODO do log file rotation
	return file
}

type contextKey string

var loggerKey = contextKey("_logger")

func FromContext(ctx context.Context) logrus.FieldLogger {
	res := ctx.Value(loggerKey)
	if res == nil {
		return logrus.StandardLogger()
	}
	return res.(logrus.FieldLogger)
}

func StashedInContext(ctx context.Context, logger logrus.FieldLogger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func logrusCopy(l *logrus.Logger) *logrus.Logger {
	hooks := make(logrus.LevelHooks)
	for level, list := range l.Hooks {
		hooks[level] = make([]logrus.Hook, len(list))
		copy(hooks[level], list)
	}
	return &logrus.Logger{
		Out:          l.Out,
		Hooks:        l.Hooks,
		Formatter:    l.Formatter,
		ReportCaller: l.ReportCaller,
		Level:        l.Level,
		ExitFunc:     l.ExitFunc,
	}
}

func ConstFields(fields Fields) logrus.Hook {
	return &fieldsHook{fields: fields}
}

type fieldsHook struct {
	fields Fields
}

func (h *fieldsHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *fieldsHook) Fire(e *logrus.Entry) error {
	for k, v := range h.fields {
		e.Data[k] = v
	}
	return nil
}

var _ logrus.Hook = (*fieldsHook)(nil)

// PrintLogger defines an interface for logging through printing.
type PrintLogger interface {
	Printf(string, ...interface{})
	Println(...interface{})
	Warning(...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
	Fatal(...interface{})
}

// ColorLogger is a logger that prints in color.
type ColorLogger struct {
	output io.Writer
	col    Color
}

var _ PrintLogger = (*ColorLogger)(nil)

// NewColorLogger creates a new logger that prints in color.
func NewColorLogger(w io.Writer, color Color) *ColorLogger {
	return &ColorLogger{
		output: w,
		col:    color,
	}
}

// Output writes a string to the logger.
func (cl *ColorLogger) Output(out string, col Color) {
	var (
		t = time.Now()
		b = &bytes.Buffer{}

		year, month, day = t.Date()
		hour, min, sec   = t.Clock()
	)

	fmt.Fprintf(b, "%s[%d/%d/%d %d:%d:%d]%s ",
		col, year, month, day, hour, min, sec, NoColor)
	fmt.Fprint(b, out)

	_, err := cl.output.Write(b.Bytes())
	if err != nil {
		fmt.Println("Logging Error: ", err)
	}
}

// Printf prints with a format
func (cl *ColorLogger) Printf(format string, v ...interface{}) {
	cl.Output(fmt.Sprintf(format, v...), cl.col)
}

// Println prints to the logger with a new line at the end.
func (cl *ColorLogger) Println(v ...interface{}) {
	cl.Output(fmt.Sprintln(v...), cl.col)
}

// Warning prints to the logger as a warning.
func (cl *ColorLogger) Warning(v ...interface{}) {
	cl.Output(fmt.Sprintln(v...), Orange)
}

// Error prints to the logger as an error.
func (cl *ColorLogger) Error(v ...interface{}) {
	cl.Output(fmt.Sprintln(v...), Red)
}

// Errorf logs a formatted error in red.
func (cl *ColorLogger) Errorf(format string, v ...interface{}) {
	cl.Output(fmt.Sprintf(format, v...), Red)
}

// Fatal will output an error and exit the program.
func (cl *ColorLogger) Fatal(v ...interface{}) {
	cl.Error(v...)
	os.Exit(1)
}
