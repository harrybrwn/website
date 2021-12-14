package app

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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

type UserStore interface {
	Find(context.Context, interface{}) (*User, error)
	Get(context.Context, uuid.UUID) (*User, error)
	Update(context.Context, *User) error
	Create(context.Context, string, *User) (*User, error)
}

func NewUserStore(db *sql.DB) *userStore {
	return &userStore{
		db: db,
	}
}

type userStore struct {
	db *sql.DB
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

func (s *userStore) Find(ctx context.Context, identifier interface{}) (*User, error) {
	var u User
	switch id := identifier.(type) {
	case uuid.UUID:
		return s.Get(ctx, id)
	case int:
		const query = selectQueryHead + `WHERE id = $1`
		row := s.db.QueryRowContext(ctx, query, id)
		err := scanUser(row, &u)
		if err != nil {
			return nil, err
		}
		return &u, nil
	default:
		const query = selectQueryHead + `WHERE email = $1 OR username = $1`
		row := s.db.QueryRowContext(ctx, query, identifier)
		err := scanUser(row, &u)
		if err != nil {
			return nil, err
		}
		return &u, nil
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

func (s *userStore) Create(ctx context.Context, password string, u *User) (*User, error) {
	var err error
	u.PWHash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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
