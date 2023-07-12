package invite

import (
	"bytes"
	"context"
	"html/template"

	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
	"gopkg.hrry.dev/homelab/pkg/email"
)

type Mailer interface {
	Send(ctx context.Context, invitation *Invitation) error
}

func NewMailer(from email.Email, subject string, t *template.Template, client email.Client) (Mailer, error) {
	if !email.Valid(from.Address) {
		return nil, email.ErrInvalid
	}
	return &mailer{
		sender:   from,
		subject:  subject,
		template: t,
		client:   client,
	}, nil
}

type mailer struct {
	sender   email.Email
	subject  string
	template *template.Template
	client   email.Client
}

func (m *mailer) Send(ctx context.Context, invitation *Invitation) error {
	if !email.Valid(invitation.Email) {
		return email.ErrInvalid
	}
	var buf bytes.Buffer
	err := m.template.Execute(&buf, invitation)
	if err != nil {
		return err
	}
	message := mail.NewSingleEmail(
		&m.sender,
		m.subject,
		&email.Email{
			Name:    invitation.ReceiverName,
			Address: invitation.Email,
		},
		"",
		buf.String(),
	)
	response, err := m.client.SendWithContext(ctx, message)
	if err != nil {
		return err
	}
	logger.WithFields(logrus.Fields{
		"status":  response.StatusCode,
		"headers": response.Headers,
		"body":    response.Body,
	}).Info("email response")
	return nil
}
