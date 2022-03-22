package chat

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"harrybrown.com/pkg/db"
	"harrybrown.com/pkg/log"
	"nhooyr.io/websocket"
)

var logger logrus.FieldLogger = logrus.StandardLogger()

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
	ID        int               `json:"id"`
	OwnerID   int               `json:"owner_id"`
	Name      string            `json:"name"`
	CreatedAt time.Time         `json:"created_at"`
	Members   []*ChatRoomMember `json:"members"`
}

type ChatRoomMember struct {
	// Room is the chatroom ID
	Room   int `json:"room"`
	UserID int `json:"user_id"`
	// LastSeen is the last message the user has seen
	LastSeen int64 `json:"last_seen"`
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
	CreateRoom(context.Context, int, string) (*Room, error)
	GetRoom(context.Context, int) (*Room, error)
	SaveMessage(ctx context.Context, msg *Message) error
	Messages(ctx context.Context, room, prev, limit int) ([]*Message, error)
}

type store struct {
	db db.DB
}

func (rs *store) GetRoom(ctx context.Context, id int) (*Room, error) {
	const query = `SELECT owner_id, name FROM chatroom WHERE id = $1`
	var r Room
	rows, err := rs.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return &r, db.ScanOne(rows, &r.OwnerID, &r.Name)
}

func (rs *store) SaveMessage(ctx context.Context, msg *Message) error {
	const query = `INSERT INTO chatroom_messages (room, user_id, body, created_at) ` +
		`VALUES ($1, $2, $3, $4)`
	_, err := rs.db.ExecContext(ctx, query, msg.Room, msg.UserID, msg.Body, msg.CreatedAt)
	return err
}

func (rs *store) CreateRoom(ctx context.Context, owner int, name string) (*Room, error) {
	const query = `INSERT INTO chatroom (owner_id, name) ` +
		`VALUES ($1, $2) RETURNING id, created_at`
	rows, err := rs.db.QueryContext(ctx, query, owner, name)
	if err != nil {
		return nil, err
	}
	var room = Room{OwnerID: owner, Name: name}
	err = db.ScanOne(rows, &room.ID, &room.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &room, nil
}

const listMessagesQuery = `
	SELECT id, room, user_id, body, created_at
	FROM chatroom_messages
	WHERE id > $1
	ORDER BY created_at DESC
	LIMIT $2`

func (rs *store) Messages(ctx context.Context, room, prev, limit int) ([]*Message, error) {
	rows, err := rs.db.QueryContext(ctx, listMessagesQuery, prev, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query")
	}
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
	}
	return msgs, nil
}

type ChatRoom struct {
	Store  Store
	RoomID int
	UserID int

	ps     PubSub
	s      Socket
	stop   chan struct{}
	logger logrus.FieldLogger
}

func OpenRoom(store Store, room, user int) *ChatRoom {
	return &ChatRoom{
		Store:  store,
		RoomID: room,
		UserID: user,
	}
}

func (cr *ChatRoom) Messages(ctx context.Context, prev, limit int) ([]*Message, error) {
	return cr.Store.Messages(ctx, cr.RoomID, prev, limit)
}

func (cr *ChatRoom) Start(ctx context.Context, pubsub PubSub, socket Socket) error {
	cr.ps = pubsub
	cr.s = socket
	cr.stop = make(chan struct{})
	cr.logger = log.FromContext(ctx)
	go cr.readLoop(ctx)
	return cr.writeLoop(ctx)
}

func (cr *ChatRoom) readLoop(ctx context.Context) error {
	for {
		msg, err := cr.s.Recv(ctx)
		if e := cr.handleSocketError(err, "failed to receive from websocket"); e != nil {
			return e
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
