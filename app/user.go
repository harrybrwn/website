package app

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"harrybrown.com/pkg/auth"
)

type User struct {
	ID        int         `json:"id"`
	UUID      uuid.UUID   `json:"uuid"`
	Username  string      `json:"username"`
	Email     string      `json:"email"`
	PWHash    []byte      `json:"-"`
	TOTPCode  string      `json:"-"`
	Roles     []auth.Role `json:"roles"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func (u *User) NewClaims() *auth.Claims {
	return &auth.Claims{
		ID:    u.ID,
		UUID:  u.UUID,
		Roles: u.Roles,
	}
}

func (u *User) VerifyPassword(pw string) error {
	return bcrypt.CompareHashAndPassword(u.PWHash, []byte(pw))
}

type UserStore interface {
	Find(context.Context, interface{}) (*User, error)
	Login(context.Context, *Login) (*User, error)
	Get(context.Context, uuid.UUID) (*User, error)
	Update(context.Context, *User) error
	Create(context.Context, string, *User) (*User, error)
}

func NewUserStore(db *sql.DB) *userStore {
	return &userStore{
		db: db,
	}
}

type Login struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *userStore) Login(ctx context.Context, l *Login) (*User, error) {
	if len(l.Password) == 0 {
		return nil, errors.New("user gave zero length password")
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
		return nil, errors.New("unable to find user")
	}
	if err != nil {
		return nil, errors.Wrap(err, "could not find user")
	}
	err = u.VerifyPassword(l.Password)
	if err != nil {
		return nil, errors.New("incorrect password")
	}
	return u, nil
}

type userStore struct {
	db     *sql.DB
	logger logrus.FieldLogger
}

const selectQueryHead = `SELECT
	id,
	uuid,
	username,
	email,
	pw_hash,
	totp_code,
	roles,
	created_at,
	updated_at
FROM "user" `

func scanUser(row *sql.Row, u *User) error {
	return row.Scan(
		&u.ID,
		&u.UUID,
		&u.Username,
		&u.Email,
		&u.PWHash,
		&u.TOTPCode,
		pq.Array(&u.Roles),
		&u.CreatedAt,
		&u.UpdatedAt,
	)
}

func (s *userStore) get(ctx context.Context, q string, args ...interface{}) (*User, error) {
	var u User
	row := s.db.QueryRowContext(ctx, q, args...)
	err := scanUser(row, &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
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

func (s *userStore) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	const query = selectQueryHead + `WHERE uuid = $1`
	var u User
	row := s.db.QueryRowContext(ctx, query, id)
	return &u, scanUser(row, &u)
}

func (s *userStore) Update(ctx context.Context, u *User) error {
	const query = `
	UPDATE "user"
	SET username = $3,
		email = $4,
		updated_at = CURRENT_TIMESTAMP
	WHERE id = $1 AND uuid = $2
	RETURNING updated_at`
	return s.db.QueryRowContext(
		ctx, query,
		u.ID,
		u.UUID,
		u.Username,
		u.Email,
	).Scan(&u.UpdatedAt)
}

const hashCost = bcrypt.DefaultCost

func (s *userStore) Create(ctx context.Context, password string, u *User) (*User, error) {
	var err error
	u.PWHash, err = bcrypt.GenerateFromPassword([]byte(password), hashCost)
	if err != nil {
		return nil, err
	}
	u.UUID = uuid.New()
	if len(u.Roles) == 0 {
		u.Roles = []auth.Role{auth.RoleDefault}
	}
	const query = `
		INSERT INTO "user" (uuid, username, email, pw_hash, roles, totp_code)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`
	row := s.db.QueryRowContext(
		ctx,
		query,
		u.UUID,
		u.Username,
		u.Email,
		u.PWHash,
		pq.Array(u.Roles),
		u.TOTPCode,
	)
	return u, row.Scan(&u.CreatedAt, &u.UpdatedAt)
}
