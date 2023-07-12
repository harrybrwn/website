package invite

import (
	"context"
	"html/template"
	"testing"
	texttemplate "text/template"

	"github.com/golang/mock/gomock"
	"github.com/matryer/is"
	"github.com/pkg/errors"
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"gopkg.hrry.dev/homelab/pkg/email"
	"gopkg.hrry.dev/homelab/pkg/internal/mocks/mockemail"
)

func TestNewMailer(t *testing.T) {
	is := is.New(t)
	m, err := NewMailer(email.Email{Address: ""}, "", nil, nil)
	is.True(errors.Is(err, email.ErrInvalid))
	is.True(m == nil)

	m, err = NewMailer(email.Email{Address: "kerry@stones.com"}, "subject", nil, nil)
	is.NoErr(err)
	ml := m.(*mailer)
	is.Equal(ml.subject, "subject")
	is.True(ml.client == nil)
}

func TestMailer_Send(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mockemail.NewMockClient(ctrl)
	mailer := mailer{
		sender:   email.Email{Name: "Jim", Address: "jim@jim.com"},
		subject:  "test email",
		template: template.Must(template.New("t").Parse("<p>{{ .TTL }}</p>")),
		client:   client,
	}
	ctx := context.Background()

	// Basic success
	client.EXPECT().SendWithContext(ctx, mail.NewV3MailInit(
		&mailer.sender,
		"test email",
		&email.Email{Address: "joe@joe.com"},
		mail.NewContent("text/html", "<p>3</p>")),
	).
		Return(&rest.Response{StatusCode: 200}, nil)
	err := mailer.Send(ctx, &Invitation{TTL: 3, Email: "joe@joe.com"})
	is.NoErr(err)

	// error from sendgrid
	client.EXPECT().SendWithContext(ctx, mail.NewV3MailInit(
		&mailer.sender,
		"test email",
		&email.Email{Address: "joe@joe.com"},
		mail.NewContent("text/html", "<p>3</p>")),
	).
		Return(nil, ErrSessionOwnership)
	err = mailer.Send(ctx, &Invitation{TTL: 3, Email: "joe@joe.com"})
	is.True(errors.Is(err, ErrSessionOwnership))

	// Bad address
	err = mailer.Send(ctx, &Invitation{Email: ""})
	is.True(errors.Is(err, email.ErrInvalid))

	// Bad template
	mailer.template = template.Must(template.New("t").Parse("{{.NoneExistantTemplateVariable}}"))
	err = mailer.Send(ctx, &Invitation{Email: "1@one.com"})
	is.True(err != nil)
	_, ok := err.(texttemplate.ExecError)
	is.True(ok)
}
