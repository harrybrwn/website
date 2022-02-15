package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"harrybrown.com/pkg/auth"
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
	Path    PathBuilder
	RDB     redis.Cmdable
	Encoder StrEncoder
}

type inviteSession struct {
	// CreatedBy is the user that created the invite.
	CreatedBy uuid.UUID `json:"cb"`
	// TTL is the number of sign-up attempts before destroying the session.
	TTL int `json:"tl"`
	// The amount of time left until the session times out.
	Timeout   time.Duration `json:"to"`
	ExpiresAt int64         `json:"ex"`
	// Force the invite to have only one valid email
	Email string `json:"e"`
}

const (
	defaultInviteTTL     = 5
	defaultInviteTimeout = time.Minute * 10
)

// Create is the handler for people with accounts to create temporary invite links
func (iv *Invitations) Create() echo.HandlerFunc {
	type Params struct {
		Timeout int    `json:"timeout"` // timeout in seconds
		TTL     int    `json:"ttl"`
		Email   string `json:"email"`
	}
	type Response struct {
		Path string `json:"path"`
	}
	return func(c echo.Context) error {
		var (
			err     error
			p       Params
			req     = c.Request()
			ctx     = req.Context()
			claims  = auth.GetClaims(c)
			timeout = defaultInviteTimeout
		)

		if claims == nil {
			logger.Error("could not find claims")
			return echo.ErrUnauthorized
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
			if p.TTL != 0 || p.Timeout != 0 {
				return echo.ErrUnauthorized
			}
		}
		if p.TTL == 0 {
			p.TTL = defaultInviteTTL
		}
		key, err := iv.key()
		if err != nil {
			return err
		}
		err = iv.put(ctx, key, &inviteSession{
			CreatedBy: claims.UUID,
			Timeout:   timeout,
			ExpiresAt: time.Now().Add(timeout).UnixMilli(),
			TTL:       p.TTL,
			Email:     p.Email,
		})
		if err != nil {
			return err
		}
		resp := Response{
			Path: filepath.Join("/", iv.Path.Path(key)),
		}
		return c.JSON(200, &resp)
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
		session, err := iv.get(ctx, id)
		if err != nil {
			return echo.ErrNotFound.SetInternal(err)
		}
		if session.TTL == 0 {
			iv.RDB.Del(ctx, id)
			return echo.ErrForbidden.SetInternal(errors.New("session ttl limit hit"))
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
		session, err := iv.get(ctx, key)
		if err != nil {
			return echo.ErrNotFound.SetInternal(err)
		}
		if session.TTL == 0 {
			_ = iv.RDB.Del(ctx, key).Err()
			return echo.ErrForbidden.SetInternal(errors.New("session ttl limit hit"))
		} else if session.TTL > 0 {
			// Decrement the TTL and put it back in storage
			session.TTL--
			err = iv.put(ctx, key, session)
			if err != nil {
				return echo.ErrInternalServerError.SetInternal(err)
			}
		}
		err = c.Bind(&login)
		if err != nil {
			return echo.ErrBadRequest.SetInternal(err)
		}
		if len(login.Email) == 0 || len(login.Password) == 0 {
			return &echo.HTTPError{Code: http.StatusBadRequest, Message: "empty login information"}
		}
		if len(session.Email) > 0 && session.Email != login.Email {
			return &echo.HTTPError{Code: http.StatusForbidden, Message: "wrong email email"}
		}
		_, err = users.Create(ctx, login.Password, &User{
			Email:    login.Email,
			Username: login.Username,
		})
		if err != nil {
			return echo.ErrInternalServerError.SetInternal(err)
		}
		// Cleanup on success
		defer iv.RDB.Del(ctx, key)
		return c.Redirect(http.StatusPermanentRedirect, "/")
	}
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
		session, err := iv.get(ctx, id)
		if err != nil {
			return echo.ErrNotFound.SetInternal(err)
		}
		if !bytes.Equal(claims.UUID[:], session.CreatedBy[:]) {
			return echo.ErrForbidden
		}
		return iv.RDB.Del(ctx, id).Err()
	}
}

func (iv *Invitations) put(
	ctx context.Context,
	key string,
	session *inviteSession,
) error {
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return iv.RDB.Set(ctx, key, raw, session.Timeout).Err()
}

func (iv *Invitations) get(ctx context.Context, key string) (*inviteSession, error) {
	var s inviteSession
	raw, err := iv.RDB.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(raw, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (iv *Invitations) key() (string, error) {
	var (
		b [32]byte
	)
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}
	return iv.Encoder.EncodeToString(b[:]), nil
}
