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

var logger = logrus.StandardLogger()

func SetLogger(l *logrus.Logger) { logger = l }

func GetLogger() *logrus.Logger {
	return logger
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
		logger.Warnf("failed to open log file: %w", err)
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
