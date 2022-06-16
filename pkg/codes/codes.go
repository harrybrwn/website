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

	InternalError  Code = http.StatusInternalServerError
	NotImplemented Code = http.StatusNotImplemented
	Unavailable    Code = http.StatusServiceUnavailable
)

func (c Code) Error() string {
	return http.StatusText(int(c))
}

func ToHTTPStatus(c Code) int {
	switch c {
	case Ok:
		return http.StatusOK
	case NoContent:
		return http.StatusNoContent

	case BadRequest:
		return http.StatusBadRequest
	case Unauthorized:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
	case NotFound:
		return http.StatusNotFound
	case MethodNotAllowed:
		return http.StatusMethodNotAllowed
	case RequestTimeout:
		return http.StatusRequestTimeout

	case InternalError:
		return http.StatusInternalServerError
	case NotImplemented:
		return http.StatusNotImplemented
	case Unavailable:
		return http.StatusServiceUnavailable

	default:
		return http.StatusInternalServerError
	}
}
