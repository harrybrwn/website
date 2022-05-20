package codes

import "net/http"

type Code int

const (
	Ok        Code = http.StatusOK
	NoContent Code = http.StatusNoContent

	BadRequest       Code = http.StatusBadRequest
	Unauthorized     Code = http.StatusUnauthorized
	Forbidden        Code = http.StatusForbidden
	NotFound         Code = http.StatusNotFound
	MethodNotAllowed Code = http.StatusMethodNotAllowed
	RequestTimeout   Code = http.StatusRequestTimeout
	Teapot           Code = http.StatusTeapot

	InternalError  Code = http.StatusInternalServerError
	NotImplemented Code = http.StatusNotImplemented
	Unavailable    Code = http.StatusServiceUnavailable
)

func (c Code) Error() string {
	return http.StatusText(int(c))
}
