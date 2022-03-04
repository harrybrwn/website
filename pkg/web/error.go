package web

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"runtime"

	"harrybrown.com/pkg/log"
)

type ErrorCode int

type Error struct {
	Status   int         `json:"-"`
	Code     ErrorCode   `json:"code"`
	Message  interface{} `json:"message"`
	Internal error       `json:"-"`
}

func (e *Error) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("code=%d, status=%d, message=%v, internal=%v", e.Code, e.Status, e.Message, e.Internal)
	}
	return fmt.Sprintf("code=%d, status=%d, message=%v", e.Code, e.Status, e.Message)
}

func (e *Error) Is(err error) bool { return errors.Is(e, err) }

func WrapError(status int, err error, message ...interface{}) error {
	e := &Error{Status: status, Internal: err}
	if len(message) > 0 {
		e.Message = message[0]
	}
	return e
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

// NotImplimented is a not implimented handler.
func NotImplimented(w http.ResponseWriter, r *http.Request) {
	ServeErrorMsg(w, "Not Implimented", http.StatusNotImplemented)
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
