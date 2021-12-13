package app

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type User struct {
	ID        int       `json:"id"`
	UUID      uuid.UUID `json:"uuid"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	PWHash    []byte    `json:"-"`
}

type UserStore interface {
	Put(context.Context, *User) (*User, error)
	Get(context.Context, int) (*User, error)
}

func NewUserStore(db *sql.DB) *userStore {
	return &userStore{
		db: db,
	}
}

type userStore struct {
	db *sql.DB
}

func (s *userStore) Put(ctx context.Context, u *User) (*User, error) {
	if u.PWHash == nil {
		return nil, errors.New("new user has no password hash")
	}
	u.UUID = uuid.New()
	if len(u.Roles) == 0 {
		u.Roles = []string{"default"}
	}
	const query = `
		INSERT INTO "user" (uuid, username, pw_hash, email, roles)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING (created_at, updated_at)`
	row := s.db.QueryRowContext(
		ctx,
		query,
		u.UUID,
		u.PWHash,
		u.Email,
		pq.Array(u.Roles),
	)
	return u, row.Scan(&u.CreatedAt, &u.UpdatedAt)
}

func (s *userStore) Get(id int) (*User, error) {
	return nil, nil
}
