package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"harrybrown.com/pkg/codes"
)

var (
	ErrUnsupportedMediaType = newStatusErr(http.StatusUnsupportedMediaType)
)

const (
	MIMEApplicationXML      = "application/xml"
	MIMEApplicationJSON     = "application/json"
	MIMEApplicationForm     = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf = "application/protobuf"
	MIMEMultipartForm       = "multipart/form-data"
	MIMEOctetStream         = "application/octet-stream"
)

func BindBody(req *http.Request, body any) error {
	if req.ContentLength == 0 {
		return nil
	}
	ct := req.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(ct, MIMEApplicationJSON):
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return wrapError(err, codes.BadRequest, "invalid body")
		}
		return wrapError(json.Unmarshal(b, body), codes.BadRequest, "invalid json body")
	case strings.HasPrefix(ct, MIMEApplicationForm):
		//err := req.ParseForm()
		//if err != nil {
		//	return wrapError(err, codes.BadRequest, "invalid form data")
		//}
		// use req.Form
		return ErrUnsupportedMediaType
	case strings.HasPrefix(ct, MIMEMultipartForm):
		//err := req.ParseMultipartForm(32 << 20) // 32 MB
		//if err != nil {
		//	return wrapError(err, codes.BadRequest, "invalid multipart form data")
		//}
		// use req.Form
		return ErrUnsupportedMediaType
	default:
		return ErrUnsupportedMediaType
	}
}

func newStatusErr(status int) error {
	return &Error{
		Status:  status,
		Message: http.StatusText(status),
	}
}

func wrapError(e error, code codes.Code, msg string) *Error {
	return &Error{
		Code:     code,
		Status:   codes.ToHTTPStatus(code),
		Message:  msg,
		Internal: e,
	}
}
