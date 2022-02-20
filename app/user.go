package app

import (
	"context"
	"database/sql"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/db"
)

type User struct {
	ID         int         `json:"id"`
	UUID       uuid.UUID   `json:"uuid"`
	Username   string      `json:"username"`
	Email      string      `json:"email"`
	PWHash     []byte      `json:"-"`
	TOTPSecret string      `json:"-"`
	Roles      []auth.Role `json:"roles"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

func (u *User) NewClaims() *auth.Claims {
	return &auth.Claims{
		ID:    u.ID,
		UUID:  u.UUID,
		Roles: u.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: []string{auth.TokenAudience},
			Issuer:   auth.Issuer,
		},
	}
}

func (u *User) VerifyPassword(pw string) error {
	return bcrypt.CompareHashAndPassword(u.PWHash, []byte(pw))
}

var (
	ErrEmptyPassword = errors.New("zero length password")
	ErrUserNotFound  = errors.New("could not find user")
	ErrWrongPassword = errors.New("password was incorrect")
)

type UserStore interface {
	Login(context.Context, *Login) (*User, error)
	Get(context.Context, uuid.UUID) (*User, error)
	Create(ctx context.Context, password string, user *User) (*User, error)
}

func NewUserStore(db db.DB) *userStore {
	return &userStore{
		db:     db,
		logger: logger,
	}
}

type userStore struct {
	db     db.DB
	logger logrus.FieldLogger
}

type Login struct {
	Username string `json:"username" form:"username"`
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}

const selectQueryHead = `SELECT
	id,
	uuid,
	username,
	email,
	pw_hash,
	totp_secret,
	roles,
	created_at,
	updated_at
FROM "user" `

func scanUser(rows db.Rows, u *User) (err error) {
	defer func() {
		e := rows.Close()
		if err == nil {
			err = e
		}
	}()
	if !rows.Next() {
		if err = rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	err = rows.Scan(
		&u.ID,
		&u.UUID,
		&u.Username,
		&u.Email,
		&u.PWHash,
		&u.TOTPSecret,
		pq.Array(&u.Roles),
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	return err
}

func (s *userStore) get(ctx context.Context, q string, args ...interface{}) (*User, error) {
	var u User
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return &u, scanUser(rows, &u)
}

func (s *userStore) Login(ctx context.Context, l *Login) (*User, error) {
	if len(l.Password) == 0 {
		return nil, ErrEmptyPassword
	}
	var (
		err error
		u   *User
	)
	if len(l.Email) > 0 {
		const query = selectQueryHead + `WHERE email = $1`
		u, err = s.get(ctx, query, l.Email)
	} else if len(l.Username) > 0 {
		const query = selectQueryHead + `WHERE username = $1`
		u, err = s.get(ctx, query, l.Username)
	} else {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "could not find user")
	}
	err = u.VerifyPassword(l.Password)
	if err != nil {
		return nil, ErrWrongPassword
	}
	return u, nil
}

func (s *userStore) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	const query = selectQueryHead + `WHERE uuid = $1`
	var u User
	rows, err := s.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	return &u, scanUser(rows, &u)
}

const hashCost = bcrypt.DefaultCost

// HashPassword using the global application hash cost.
func HashPassword(pw []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(pw, hashCost)
}

const createUserQuery = `
	INSERT INTO "user" (uuid, username, email, pw_hash, roles, totp_secret)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, created_at, updated_at`

func (s *userStore) Create(ctx context.Context, password string, u *User) (*User, error) {
	var err error
	u.PWHash, err = HashPassword([]byte(password))
	if err != nil {
		return nil, err
	}
	u.UUID = uuid.New()
	if len(u.Roles) == 0 {
		u.Roles = []auth.Role{auth.RoleDefault}
	}
	rows, err := s.db.QueryContext(
		ctx,
		createUserQuery,
		u.UUID,
		u.Username,
		u.Email,
		u.PWHash,
		pq.Array(u.Roles),
		u.TOTPSecret,
	)
	if err != nil {
		return nil, err
	}
	return u, db.ScanOne(rows, &u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (s *userStore) Update(ctx context.Context, u *User) error {
	const query = `
	UPDATE "user"
	SET username = $3,
		email = $4,
		updated_at = CURRENT_TIMESTAMP
	WHERE id = $1 AND uuid = $2
	RETURNING updated_at`
	rows, err := s.db.QueryContext(
		ctx, query,
		u.ID,
		u.UUID,
		u.Username,
		u.Email,
	)
	if err != nil {
		return err
	}
	return db.ScanOne(rows, &u.UpdatedAt)
}

func (s *userStore) Find(ctx context.Context, identifier interface{}) (*User, error) {
	switch id := identifier.(type) {
	case uuid.UUID:
		return s.Get(ctx, id)
	case int:
		return s.get(ctx, selectQueryHead+`WHERE id = $1`, id)
	default:
		const query = selectQueryHead + `WHERE email = $1 OR username = $1`
		return s.get(ctx, query, identifier)
	}
}
