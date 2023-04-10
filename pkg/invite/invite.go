package invite

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/log"
)

var logger = log.GetLogger()

const (
	defaultInviteTTL     = 5
	defaultInviteTimeout = time.Minute * 10
)

var (
	ErrInviteTTL        = errors.New("session ttl limit reached")
	ErrSessionOwnership = errors.New("cannot access session created by someone else")
)

type Session struct {
	// CreatedBy is the user that created the invite.
	CreatedBy uuid.UUID `json:"cb,omitempty"`
	// TTL is the number of sign-up attempts before destroying the session.
	TTL int `json:"tl,omitempty"`
	// ExpiresAt is the timestamp unix millisecond timestamp at which the
	// session expires.
	ExpiresAt int64 `json:"ex,omitempty"`
	// Force the invite to have only one valid email
	Email string `json:"e,omitempty"`
	// Roles is an array of roles used when creating the new user. Only Admin
	// should be able to set roles.
	Roles []auth.Role `json:"r,omitempty"`
	// Not actually stored in session, used as metadata
	ID string `json:"-"`
}

type Invitation struct {
	Path         string      `json:"path"`
	CreatedBy    uuid.UUID   `json:"created_by"`
	ExpiresAt    time.Time   `json:"expires_at"`
	Email        string      `json:"email,omitempty"`
	ReceiverName string      `json:"receiver_name,omitempty"`
	Roles        []auth.Role `json:"roles"`
	TTL          int         `json:"ttl"`

	Domain string `json:"-"`
}

type CreateInviteRequest struct {
	Timeout      time.Duration `json:"timeout,omitempty"`
	TTL          int           `json:"ttl,omitempty"`
	Email        string        `json:"email,omitempty"`
	ReceiverName string        `json:"receiver_name,omitempty"`
	Roles        []string      `json:"roles"`
}

type Store interface {
	Create(context.Context, uuid.UUID, *CreateInviteRequest) (*Session, string, error)
	Get(context.Context, string) (*Session, error)
	View(context.Context, string) (*Session, error)
	OwnerDel(context.Context, string, uuid.UUID) error
	Del(context.Context, string) error
	List(ctx context.Context) ([]*Session, error)
}

func NewStore(rdb redis.Cmdable, prefix string) Store {
	return &SessionStore{
		RDB:    rdb,
		Prefix: prefix,
		KeyGen: DefaultKeyGen,
		Now:    time.Now,
	}
}

type SessionStore struct {
	RDB    redis.Cmdable
	Prefix string
	Now    func() time.Time
	KeyGen func() (string, error)
}

func (ss *SessionStore) key(id string) string {
	return fmt.Sprintf("%s:%s", ss.Prefix, id)
}

func (ss *SessionStore) Create(ctx context.Context, creator uuid.UUID, req *CreateInviteRequest) (*Session, string, error) {
	var (
		timeout = req.Timeout
		ttl     = req.TTL
	)
	if req.Timeout == 0 {
		timeout = defaultInviteTimeout
	}
	if req.TTL == 0 {
		ttl = defaultInviteTTL
	}
	if ss.Now == nil {
		ss.Now = time.Now
	}

	s := Session{
		CreatedBy: creator,
		ExpiresAt: ss.Now().Add(timeout).UnixMilli(),
		TTL:       ttl,
		Email:     req.Email,
		Roles:     asAuthRoles(req.Roles),
	}
	raw, err := json.Marshal(&s)
	if err != nil {
		return nil, "", err
	}
	id, err := ss.KeyGen()
	if err != nil {
		return nil, "", err
	}
	err = ss.RDB.Set(ctx, ss.key(id), raw, timeout).Err()
	if err != nil {
		return nil, "", err
	}
	return &s, id, nil
}

func DefaultKeyGen() (string, error) {
	var b [32]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", errors.Wrap(err, "failed to generate session id")
	}
	id := base64.RawURLEncoding.EncodeToString(b[:])
	return id, nil
}

func (ss *SessionStore) Get(ctx context.Context, key string) (*Session, error) {
	s, err := ss.View(ctx, key)
	if err != nil {
		return nil, err
	}
	if s.TTL > 0 {
		s.TTL--
		err = ss.put(ctx, key, s)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (ss *SessionStore) View(ctx context.Context, key string) (*Session, error) {
	s, err := ss.get(ctx, key)
	if err != nil {
		return nil, err
	}
	if s.TTL == 0 {
		err = ss.Del(ctx, key)
		if err != nil {
			logger.WithError(err).Error("could not delete invite expired session")
		}
		return nil, ErrInviteTTL
	}
	return s, nil
}

func (ss *SessionStore) put(ctx context.Context, key string, session *Session) error {
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return ss.RDB.Set(ctx, ss.key(key), raw, redis.KeepTTL).Err()
}

func (ss *SessionStore) OwnerDel(ctx context.Context, key string, uid uuid.UUID) error {
	s, err := ss.get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return echo.ErrNotFound
		}
		return echo.ErrInternalServerError.SetInternal(err)
	}
	if !bytes.Equal(s.CreatedBy[:], uid[:]) {
		return echo.ErrForbidden.SetInternal(ErrSessionOwnership)
	}
	err = ss.Del(ctx, key)
	if err != nil {
		return echo.ErrInternalServerError.SetInternal(err)
	}
	return nil
}

func (ss *SessionStore) Del(ctx context.Context, key string) error {
	return ss.RDB.Del(ctx, ss.key(key)).Err()
}

func (ss *SessionStore) get(ctx context.Context, key string) (*Session, error) {
	raw, err := ss.RDB.Get(ctx, ss.key(key)).Bytes()
	if err != nil {
		return nil, err
	}
	var s Session
	if err = json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (ss *SessionStore) List(ctx context.Context) ([]*Session, error) {
	keys, err := ss.RDB.Keys(ctx, fmt.Sprintf("%s:*", ss.Prefix)).Result()
	if err != nil {
		return nil, err
	}
	l := len(keys)
	if l == 0 {
		return []*Session{}, nil
	}
	sessions := make([]*Session, l)
	rawsessions, err := ss.RDB.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	for i, raw := range rawsessions {
		s := raw.(string)
		sessions[i] = new(Session)
		err = json.Unmarshal([]byte(s), sessions[i])
		if err != nil {
			return nil, err
		}
		ix := strings.LastIndex(keys[i], ":")
		if ix >= 0 {
			sessions[i].ID = keys[i][ix+1:]
		}
	}
	return sessions, nil
}

func asAuthRoles(ss []string) []auth.Role {
	roles := make([]auth.Role, len(ss))
	for i, s := range ss {
		roles[i] = auth.ParseRole(s)
	}
	return roles
}
