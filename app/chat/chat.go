package chat

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"harrybrown.com/pkg/db"
	"harrybrown.com/pkg/log"
	"nhooyr.io/websocket"
)

var logger logrus.FieldLogger = log.GetLogger()

func SetLogger(l logrus.FieldLogger) { logger = l }

type MsgDeviveryStatus int

const (
	// StatusSent marks when a message has been sent directly to another person and
	// stored in long-term storage.
	StatusSent MsgDeviveryStatus = iota
	// StatusStored marks when a message has been stored in long-term storage but
	// not delivered to anyone directly.
	StatusStored
	// StatusFailed marks when a message has not been stored or sent.
	StatusFailed
)

type MsgType int

const (
	// MsgEmpty is an empty message. Used for healthchecks.
	MsgEmpty MsgType = iota
	// MsgChat is a message with a chat body
	MsgChat
)

type Room struct {
	ID        int                     `json:"id" db:"id"`
	OwnerID   int                     `json:"owner_id" db:"owner_id"`
	Name      string                  `json:"name" db:"name"`
	Public    bool                    `json:"public" db:"public"`
	CreatedAt time.Time               `json:"created_at"`
	Members   map[int]*ChatRoomMember `json:"members"`
}

type ChatRoomMember struct {
	// Room is the chatroom ID
	Room   int `json:"room"`
	UserID int `json:"user_id"`
	// LastSeen is the last message the user has seen
	LastSeen int64  `json:"last_seen"`
	Username string `json:"username"`
}

type Message struct {
	ID        int       `json:"id"`
	Room      int       `json:"room"`
	UserID    int       `json:"user_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func NewStore(db db.DB) Store {
	return &store{db: db}
}

type Store interface {
	CreateRoom(ctx context.Context, owner int, name string, public bool) (*Room, error)
	GetRoom(ctx context.Context, roomID int) (*Room, error)
	SaveMessage(ctx context.Context, msg *Message) error
	Messages(ctx context.Context, room int, opts db.PaginationOpts) ([]*Message, error)
}

type store struct {
	db db.DB
}

const roomMembersQuery = `
	SELECT m.user_id, m.last_seen, u.username
	FROM chatroom_members m
	LEFT JOIN "user" u ON (m.user_id = u.id)
	WHERE m.room = $1`

func (rs *store) GetRoom(ctx context.Context, id int) (*Room, error) {
	const query = `SELECT owner_id, name FROM chatroom WHERE id = $1`
	var r = Room{
		ID:      id,
		Members: make(map[int]*ChatRoomMember),
	}
	rows, err := rs.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	err = db.ScanOne(rows, &r.OwnerID, &r.Name)
	if err != nil {
		return nil, err
	}
	// TODO use json_agg to get members as a json blob and then decode instead
	// of sending a second query.
	memberRows, err := rs.db.QueryContext(ctx, roomMembersQuery, id)
	if err != nil {
		return nil, err
	}
	defer memberRows.Close()
	for memberRows.Next() {
		var member = &ChatRoomMember{Room: id}
		err = memberRows.Scan(&member.UserID, &member.LastSeen, &member.Username)
		if err != nil {
			return nil, err
		}
		r.Members[member.UserID] = member
	}
	return &r, nil
}

func (rs *store) SaveMessage(ctx context.Context, msg *Message) error {
	const query = `INSERT INTO chatroom_messages (room, user_id, body, created_at) ` +
		`VALUES ($1, $2, $3, $4) RETURNING id`
	if msg == nil {
		return errors.New("cannot save nil message")
	}
	rows, err := rs.db.QueryContext(ctx, query, msg.Room, msg.UserID, msg.Body, msg.CreatedAt)
	if err != nil {
		return err
	}
	return db.ScanOne(rows, &msg.ID)
}

func (rs *store) CreateRoom(ctx context.Context, owner int, name string, public bool) (*Room, error) {
	const query = `INSERT INTO chatroom (owner_id,name,public) VALUES ($1,$2,$3) RETURNING id, created_at`
	rows, err := rs.db.QueryContext(ctx, query, owner, name, public)
	if err != nil {
		return nil, err
	}
	var room = Room{
		OwnerID: owner,
		Name:    name,
		Members: make(map[int]*ChatRoomMember),
		Public:  public,
	}
	err = db.ScanOne(rows, &room.ID, &room.CreatedAt)
	if err != nil {
		return nil, err
	}
	const memberQuery = `INSERT INTO chatroom_members (room,user_id) VALUES ($1,$2)`
	_, err = rs.db.ExecContext(ctx, memberQuery, room.ID, owner)
	if err != nil {
		return nil, err
	}
	room.Members[owner] = &ChatRoomMember{Room: room.ID, UserID: owner}
	return &room, nil
}

const (
	listMessagesQueryHead = `
	SELECT id, room, user_id, body, created_at
	FROM   chatroom_messages`
	listMessagesQueryOffset = listMessagesQueryHead + `
	WHERE  room = $1
	ORDER  BY created_at DESC
	LIMIT  $2 OFFSET $3`
	listMessagesQueryIDs = listMessagesQueryHead + `
	WHERE  room = $1 AND id < $2
	ORDER  BY created_at DESC
	LIMIT  $3`
)

func (rs *store) Messages(ctx context.Context, room int, opts db.PaginationOpts) ([]*Message, error) {
	var (
		err  error
		rows db.Rows
	)
	if opts.Offset == 0 && opts.Prev != 0 {
		rows, err = rs.db.QueryContext(ctx, listMessagesQueryIDs, room, opts.Prev, opts.Limit)
	} else {
		rows, err = rs.db.QueryContext(ctx, listMessagesQueryOffset, room, opts.Limit, opts.Offset)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query")
	}
	defer rows.Close()
	msgs := make([]*Message, 0)
	for rows.Next() {
		var msg Message
		err = rows.Scan(
			&msg.ID,
			&msg.Room,
			&msg.UserID,
			&msg.Body,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan message from database")
		}
		msgs = append(msgs, &msg)
	}
	return msgs, nil
}

type ChatRoom struct {
	Store  Store
	RDB    redis.UniversalClient
	RoomID int
	UserID int

	ps     PubSub
	s      Socket
	stop   chan struct{}
	logger logrus.FieldLogger
}

func OpenRoom(
	store Store,
	rdb redis.UniversalClient,
	room, user int,
) *ChatRoom {
	return &ChatRoom{
		Store:  store,
		RDB:    rdb,
		RoomID: room,
		UserID: user,
	}
}

func (cr *ChatRoom) Messages(ctx context.Context, opts db.PaginationOpts) ([]*Message, error) {
	return cr.Store.Messages(ctx, cr.RoomID, opts)
}

func (cr *ChatRoom) Exists(ctx context.Context) error {
	room, err := cr.Store.GetRoom(ctx, cr.RoomID)
	if err != nil {
		return err
	}
	if room.Public {
		return nil
	}
	if _, ok := room.Members[cr.UserID]; !ok {
		return errors.New("user is not a member of this room")
	}
	return nil
}

func (cr *ChatRoom) Start(ctx context.Context, pubsub PubSub, socket Socket) error {
	cr.ps = pubsub
	cr.s = socket
	cr.stop = make(chan struct{})
	cr.logger = log.FromContext(ctx)
	go cr.readLoop(ctx)
	return cr.writeLoop(ctx)
}

func (cr *ChatRoom) readLoop(ctx context.Context) {
	for {
		msg, err := cr.s.Recv(ctx)
		if err != nil {
			cr.handleSocketError(err, "failed to receive from websocket")
			return
		}
		err = cr.Store.SaveMessage(ctx, msg)
		if err != nil {
			cr.logger.WithError(err).Error("could not write new message to database")
			continue
		}
		err = cr.ps.Pub(ctx, msg)
		if err != nil {
			cr.logger.WithError(err).Error("could not publish message")
			continue
		}
	}
}

func (cr *ChatRoom) writeLoop(ctx context.Context) error {
	messages := cr.ps.Sub(ctx)
	for {
		select {
		case msg := <-messages:
			err := cr.s.Send(ctx, &msg)
			if e := cr.handleSocketError(err, "failed to send through websocket"); e != nil {
				return e
			}
		case <-ctx.Done():
			cr.logger.WithError(ctx.Err()).Warn("context cancelled")
			close(cr.stop)
			return nil
		case <-cr.stop:
			cr.logger.Info("stopping")
			return nil
		}
	}
}

func (cr *ChatRoom) handleSocketError(e error, message string) error {
	switch websocket.CloseStatus(e) {
	case websocket.StatusGoingAway:
		close(cr.stop)
		return nil
	default:
		if e != nil {
			cr.logger.WithError(e).Error(message)
			return errors.Wrap(e, message)
		}
	}
	return nil
}
