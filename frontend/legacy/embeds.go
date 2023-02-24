package frontend

import _ "embed"

var (
	//go:embed pages/404.html
	NotFoundHTML []byte
	//go:embed pages/invite_email.html
	InviteEmailHTML []byte
)
