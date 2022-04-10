package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/email"
	"harrybrown.com/pkg/invite"
)

var (
	ErrEmptyLogin           = &echo.HTTPError{Code: http.StatusBadRequest, Message: "empty login information"}
	ErrInviteEmailMissmatch = &echo.HTTPError{Code: http.StatusForbidden, Message: "email does not match invitation"}
	ErrInvalidTimeout       = &echo.HTTPError{Code: http.StatusBadRequest, Message: "invalid invite timeout"}
)

type StrEncoder interface {
	EncodeToString([]byte) string
}

type PathBuilder interface {
	// Return a url path
	Path(id string) string
	GetID(*http.Request) string
}

type Invitations struct {
	Path   PathBuilder
	Mailer invite.Mailer
	store  invite.Store
}

func NewInvitations(rdb redis.Cmdable, path PathBuilder, mailer invite.Mailer) *Invitations {
	return &Invitations{
		Path:  path,
		store: invite.NewStore(rdb, "invite"),
	}
}

const (
	defaultInviteTTL     = 5
	defaultInviteTimeout = time.Minute * 10
)

// Create is the handler for people with accounts to create temporary invite links
func (iv *Invitations) Create() echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			err    error
			p      invite.CreateInviteRequest
			req    = c.Request()
			ctx    = req.Context()
			claims = auth.GetClaims(c)
		)

		if claims == nil {
			return echo.ErrUnauthorized.SetInternal(auth.ErrNoClaims)
		}

		// Read the params
		err = json.NewDecoder(req.Body).Decode(&p)
		if err != io.EOF && err != nil {
			req.Body.Close()
			logger.WithError(err).Error("could not parse json params")
			return err
		}
		req.Body.Close()

		if !auth.IsAdmin(claims) {
			// Disallow these parameters if the user is not an admin.
			if p.TTL != 0 || p.Timeout != 0 || len(p.Roles) > 0 {
				return echo.ErrUnauthorized.SetInternal(auth.ErrAdminRequired)
			}
		} else {
			if p.Timeout < 0 {
				return ErrInvalidTimeout
			}
			logger.WithFields(logrus.Fields{
				"ttl":     p.TTL,
				"timeout": fmt.Sprintf("%v", p.Timeout),
			}).Debug("admin creating invite")
		}

		session, key, err := iv.store.Create(ctx, claims.UUID, &p)
		if err != nil {
			return echo.ErrInternalServerError.SetInternal(err)
		}
		inv := invite.Invitation{
			Path:         filepath.Join("/", iv.Path.Path(key)),
			ExpiresAt:    time.UnixMilli(session.ExpiresAt),
			CreatedBy:    claims.UUID,
			TTL:          session.TTL,
			Roles:        session.Roles,
			Email:        session.Email,
			ReceiverName: p.ReceiverName,
			Domain:       Domain,
		}

		if iv.Mailer != nil && email.Valid(inv.Email) {
			err = iv.Mailer.Send(ctx, &inv)
			if err != nil {
				return &echo.HTTPError{
					Code:     http.StatusInternalServerError,
					Message:  "failed to send email invite",
					Internal: err,
				}
			}
		}
		return c.JSON(200, &inv)
	}
}

func (iv *Invitations) Accept(body []byte, contentType string) echo.HandlerFunc {
	template, err := template.New("invitation-accept").Parse(string(body))
	if err != nil {
		// will happen at startup
		panic(err)
	}
	type TemplateData struct {
		// Email is the only email that will be accepted for the new account.
		Email string
		// ExpiresAt is the UNIX millisecond epoch timestamp for the session
		// expiration.
		ExpiresAt int64
		Path      string
		TriesLeft int
	}
	return func(c echo.Context) error {
		var (
			req = c.Request()
			ctx = req.Context()
			id  = iv.Path.GetID(req)
		)
		session, err := iv.store.View(ctx, id)
		if err != nil {
			return echo.ErrNotFound.SetInternal(err)
		}
		if session.ExpiresAt < 0 {
			return echo.ErrForbidden
		}
		resp := c.Response()
		resp.Header().Set("Content-Type", contentType)
		resp.WriteHeader(200)
		err = template.Execute(resp, &TemplateData{
			Email:     session.Email,
			ExpiresAt: session.ExpiresAt,
			Path:      iv.Path.Path(id),
			TriesLeft: session.TTL,
		})
		if err != nil {
			return echo.ErrInternalServerError.SetInternal(err)
		}
		return nil
	}
}

func (iv *Invitations) SignUp(users UserStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			req   = c.Request()
			ctx   = req.Context()
			key   = iv.Path.GetID(req)
			login Login
		)
		session, err := iv.store.Get(ctx, key)
		if err != nil {
			switch err {
			case invite.ErrInviteTTL:
				return echo.ErrForbidden.SetInternal(err)
			case redis.Nil:
				return echo.ErrNotFound.SetInternal(err)
			}
			return echo.ErrInternalServerError.SetInternal(err)
		}

		err = c.Bind(&login)
		if err != nil {
			return echo.ErrBadRequest.SetInternal(err)
		}
		if len(login.Email) == 0 || len(login.Password) == 0 {
			return ErrEmptyLogin
		}
		if len(session.Email) > 0 && session.Email != login.Email {
			return ErrInviteEmailMissmatch
		}
		_, err = users.Create(ctx, login.Password, &User{
			Email:    login.Email,
			Username: login.Username,
			Roles:    session.Roles,
		})
		if err != nil {
			return echo.ErrInternalServerError.SetInternal(err)
		}
		// Cleanup on success
		err = iv.store.Del(ctx, key)
		if err != nil {
			logger.WithError(err).Error("failed to destroy invite session")
		}
		return nil
	}
}

type inviteList struct {
	Invites []invite.Invitation `json:"invites"`
}

func (iv *Invitations) List() echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			resp   inviteList
			ctx    = c.Request().Context()
			claims = auth.GetClaims(c)
		)
		if claims == nil {
			return echo.ErrUnauthorized
		}
		sessions, err := iv.store.List(ctx)
		if err != nil {
			return echo.ErrInternalServerError.SetInternal(err)
		}

		resp.Invites = make([]invite.Invitation, 0, len(sessions))
		if auth.IsAdmin(claims) {
			for _, s := range sessions {
				inv := invite.Invitation{Path: iv.Path.Path(s.ID)}
				setInviteFromSession(&inv, s)
				resp.Invites = append(resp.Invites, inv)
			}
		} else {
			for _, s := range sessions {
				if !bytes.Equal(s.CreatedBy[:], claims.UUID[:]) {
					continue
				}
				inv := invite.Invitation{Path: iv.Path.Path(s.ID)}
				setInviteFromSession(&inv, s)
				resp.Invites = append(resp.Invites, inv)
			}
		}
		return c.JSON(200, resp)
	}
}

func setInviteFromSession(inv *invite.Invitation, s *invite.Session) {
	inv.Email = s.Email
	inv.CreatedBy = s.CreatedBy
	inv.ExpiresAt = time.UnixMilli(s.ExpiresAt)
	inv.Roles = s.Roles
	inv.TTL = s.TTL
}

func (iv *Invitations) Delete() echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			ctx    = c.Request().Context()
			id     = c.Param("id")
			claims = auth.GetClaims(c)
		)
		if claims == nil {
			return echo.ErrUnauthorized
		}
		return iv.store.OwnerDel(ctx, id, claims.UUID)
	}
}
