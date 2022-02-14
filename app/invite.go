package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"harrybrown.com/pkg/auth"
)

type StrEncoder interface {
	EncodeToString([]byte) string
}

type PathBuilder interface {
	// Return a url path
	Path(id string) string
}

type pathBuilder func(string) string

func (pb pathBuilder) Path(id string) string {
	return pb(id)
}

type Invitations struct {
	Path    PathBuilder
	RDB     redis.Cmdable
	Users   UserStore
	Encoder StrEncoder
}

type inviteSession struct {
	CreatedBy uuid.UUID `json:"cb"`
	// TTL is the number of times the temporary url can be visited before it
	// self destructs.
	TTL int `json:"ttl"`
	// The amount of time left until the session times out.
	Timeout time.Duration `json:"to"`
}

func (iv *Invitations) Create() echo.HandlerFunc {
	type Params struct {
		Timeout int `json:"timeout" url:"timeout"` // timeout in seconds
		TTL     int `json:"ttl"`
	}
	type Response struct {
		Path string `json:"path"`
	}
	return func(c echo.Context) error {
		var (
			err error
			p   Params

			req    = c.Request()
			ctx    = req.Context()
			claims = auth.GetClaims(c)

			timeout = time.Hour
			ttl     = 10
		)
		err = json.NewDecoder(req.Body).Decode(&p)
		if err != nil {
			return err
		}
		key, err := iv.key()
		if err != nil {
			return err
		}
		err = iv.put(ctx, key, &inviteSession{
			CreatedBy: claims.UUID,
			Timeout:   timeout,
			TTL:       ttl,
		})
		if err != nil {
			return err
		}
		resp := Response{}
		if iv.Path != nil {
			resp.Path = filepath.Join("/", iv.Path.Path(key))
		} else {
			resp.Path = filepath.Join("/", key)
		}
		return c.JSON(200, &resp)
	}
}

func (iv *Invitations) Accept() echo.HandlerFunc {
	return func(c echo.Context) error {
		return nil
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
