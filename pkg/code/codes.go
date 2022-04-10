package code

import "net/http"

type Code int

const (
	Forbidden Code = http.StatusForbidden
	NotFound  Code = http.StatusNotFound

	InternalError Code = http.StatusInternalServerError
)
