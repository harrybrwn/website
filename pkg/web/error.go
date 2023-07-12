package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"runtime"

	"github.com/labstack/echo/v4"
	"gopkg.hrry.dev/homelab/pkg/codes"
	"gopkg.hrry.dev/homelab/pkg/log"
)

type ErrorCode int

type Error struct {
	// Application specific error code
	Code codes.Code `json:"code,omitempty"`
	// Error message
	Message any `json:"message"`
	// HTTP Status code
	Status int `json:"-"`
	// Internal error and should not be known to caller
	Internal error `json:"-"`
}

func (e *Error) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("code=%d status=%d message=%v internal=%v", e.Code, e.Status, e.Message, e.Internal)
	}
	return fmt.Sprintf("code=%d status=%d message=%v", e.Code, e.Status, e.Message)
}

func (e *Error) Is(err error) bool {
	if e == err {
		return true
	}
	switch er := err.(type) {
	case nil:
		return false
	case *Error:
		return (e.Status == er.Status && errors.Is(e.Internal, er.Internal)) ||
			(e.Status == er.Status && e.Message == er.Message)
	}
	return errors.Is(e, err)
}

func StatusError(status int, err error, message ...interface{}) error {
	e := &Error{Status: status, Internal: err}
	l := len(message)
	if l >= 1 {
		e.Message = message[0]
		// TODO for l > 1 do a strings.Join for interfaces to capture all messages
	}
	return e
}

// WrapError will wrap an error in a web.Error and optionally set the message.
func WrapError(err error, message ...any) *Error {
	e := Error{Internal: err, Message: nil}
	l := len(message)
	if l > 0 {
		e.Message = message[0]
	}
	switch er := err.(type) {
	case nil:
		return nil
	case codes.Code:
		e.Code = er
		e.Status = codes.ToHTTPStatus(er)
	case *Error:
		if e.Message == nil {
			e.Message = er.Message
		} else {
			e.Message = fmt.Sprintf("%v, %v", er.Message, er.Message)
		}
		e.Code = er.Code
		e.Status = er.Status
	default:
		e.Status = http.StatusInternalServerError
	}
	return &e
}

func WriteError(w http.ResponseWriter, err error) {
	fields := log.Fields{"error": err.Error()}
	if _, file, line, ok := runtime.Caller(1); ok {
		fields["file"] = file
		fields["line"] = line
	}
	logger := logger.WithFields(fields)

	switch e := err.(type) {
	case nil:
		logger.Warn("attempted to write an nil error to http response")
		return
	case codes.Code:
		w.WriteHeader(int(e))
		_, err := w.Write([]byte(e.Error()))
		if err != nil {
			logger.WithFields(log.Fields{
				"error_response": e,
			}).Error("failed to write error_response")
		}
		statusLog(int(e), logger.WithFields(log.Fields{
			"status": int(e),
		}), "sending error response")
	case *Error:
		err := writeErrorAsJSON(logger, w, e)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.WithFields(log.Fields{
				"error_response": e,
			}).Error("failed to write error_response")
			return
		}
	case *echo.HTTPError:
		message := echo.Map{"message": e.Message}
		raw, err := json.Marshal(message)
		if err != nil {
			logger.WithFields(log.Fields{
				"error_response": e,
			}).Error("failed to write error_response")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(e.Code)
		w.Write(raw)
		statusLog(e.Code, logger.WithFields(log.Fields{
			"message":  e.Message,
			"status":   e.Code,
			"internal": e.Internal,
		}), "sending error response")
	case *json.MarshalerError:
		status := http.StatusInternalServerError
		w.WriteHeader(status)
		statusLog(status, logger.WithFields(log.Fields{
			"status": status,
		}), "failed to marshal json")
	default:
		status := http.StatusInternalServerError
		w.WriteHeader(status)
		statusLog(status, logger.WithFields(log.Fields{
			"status": status,
		}), "sending error response")
	}
}

// ErrorStatusCode will infer the http status code given an error.
func ErrorStatusCode(err error) int {
	switch e := err.(type) {
	case nil:
		return http.StatusOK
	case codes.Code:
		return int(e)
	case *Error:
		if e.Status == 0 {
			return codes.ToHTTPStatus(e.Code)
		}
		return e.Status
	case *echo.HTTPError:
		return e.Code
	default:
		return http.StatusInternalServerError
	}
}

func writeErrorAsJSON(logger log.FieldLogger, w http.ResponseWriter, e *Error) error {
	raw, err := json.Marshal(e)
	if err != nil {
		return err
	}
	status := e.Status
	if status == 0 {
		status = codes.ToHTTPStatus(e.Code)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(raw)
	if err != nil {
		return err
	}
	statusLog(status, logger.WithFields(log.Fields{
		"message":  e.Message,
		"code":     int(e.Code),
		"status":   e.Status,
		"internal": e.Internal,
	}), "sending error response")
	return nil
}

func statusLog(status int, l log.FieldLogger, msg string) {
	if status < 400 {
		// OK and Redirects
		l.Info(msg)
	} else if status < 500 {
		// 4xx errors
		l.Warn(msg)
	} else if status >= 500 {
		// 5xx errors
		l.Error(msg)
	}
}

// ErrorHandler is an error type for internal website errors.
type ErrorHandler struct {
	msg      string
	status   int
	file     string
	funcname string
	line     int
}

// Errorf create an error with a formatted message.
func Errorf(status int, format string, vars ...interface{}) error {
	pc, file, line, _ := runtime.Caller(1)

	e := &ErrorHandler{
		msg:      fmt.Sprintf(format, vars...),
		status:   status,
		file:     file,
		line:     line,
		funcname: runtime.FuncForPC(pc).Name(),
	}
	return e
}

func (h *ErrorHandler) Error() string {
	return fmt.Sprintf("(%s:%d %s()) %s\n", h.file, h.line, h.funcname, h.msg)
}

func (h *ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ServeError(w, h.status)
}

var (
	_ error        = (*ErrorHandler)(nil)
	_ http.Handler = (*ErrorHandler)(nil)
)

var errorHTML = `<!DOCTYPE html>
<html lang="en">

<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<meta http-equiv="X-UA-Compatible" content="ie=edge">
	<title>{{.Title}}</title>
<style>h1, .ErrorMsg { text-align: center; }</style>
</head>
<body>
	<div class="container">
	<h1>Response Code {{.Status}}</h1>
	<div class="ErrorMsg">
		<p>{{.Msg}}</p>
	</div>
</div>
</body>
</html>`

// ServeError serves a generic http error page.
func ServeError(w http.ResponseWriter, status int) {
	ServeErrorMsg(w, "Sorry, I must have broken something.", status)
}

// NotFound returns a not found page.
func NotFound(w http.ResponseWriter, r *http.Request) {
	ServeErrorMsg(w, "Not Found", 404)
}

// NotImplimented is a not implemented handler.
func NotImplemented(w http.ResponseWriter, r *http.Request) {
	ServeErrorMsg(w, "Not Implemented", http.StatusNotImplemented)
}

// ServeErrorMsg will serve a webpage displaying the error message and status code.
func ServeErrorMsg(w http.ResponseWriter, msg string, status int) {
	w.WriteHeader(status)
	t, err := template.New("err").Parse(errorHTML)
	if err != nil {
		log.Error("Error when serving error page:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err = t.ExecuteTemplate(w, "err", struct {
		Title, Msg string
		Status     int
	}{
		Title:  "Error",
		Msg:    msg,
		Status: status,
	}); err != nil {
		log.Error("Error when serving error page:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
