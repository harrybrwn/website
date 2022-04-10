package email

import (
	"context"
	"errors"
	"regexp"

	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

var ErrInvalid = errors.New("invalid email")

type Client interface {
	SendWithContext(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error)
}

type Email = mail.Email

var emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// Valid returns true when passed a valid email address.
func Valid(address string) bool {
	if len(address) == 0 {
		return false
	}
	return emailRegexp.MatchString(address)
}
