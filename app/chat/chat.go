package chat

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"harrybrown.com/pkg/db"
	"nhooyr.io/websocket"
)

var logger logrus.FieldLogger

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
	ID      int    `json:"id"`
	OwnerID int    `json:"owner_id"`
	Name    string `json:"name"`
}

type ChatRoomMember struct {
	// Room is the chatroom ID
	Room   int `json:"room"`
	UserID int `json:"user_id"`
	// LastSeen is the last message the user has seen
	LastSeen int64 `json:"last_seen"`
}

type ChatRoomMessage struct {
	Room      int       `json:"room"`
	UserID    int       `json:"user_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

func NewStore(db db.DB) Store {
	return &store{db: db}
}

type Store interface {
	GetChatRoom(context.Context, int) (*Room, error)
	SaveMessage(ctx context.Context, msg *ChatRoomMessage) error
}

type store struct {
	db db.DB
}

func (rs *store) GetChatRoom(ctx context.Context, id int) (*Room, error) {
	const query = `SELECT owner_id, name WHERE id = $1`
	var r Room
	rows, err := rs.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return &r, db.ScanOne(rows, &r.OwnerID, &r.Name)
}

func (rs *store) SaveMessage(ctx context.Context, msg *ChatRoomMessage) error {
	const query = `INSERT INTO chatroom_messages (room, user_id, message) VALUES ($1, $2, $3)`
	_, err := rs.db.ExecContext(ctx, query, msg.Room, msg.UserID, msg.Message)
	return err
}

func EchoHandler(w http.ResponseWriter, r *http.Request) error {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		return err
	}
	defer c.Close(websocket.StatusInternalError, "closing socket")
	l := rate.NewLimiter(rate.Every(time.Millisecond*100), 10)
	ctx := r.Context()
	for {
		err = Echo(ctx, c, l)
		if err == nil {
			continue
		}
		switch websocket.CloseStatus(err) {
		case websocket.StatusNormalClosure:
			return nil
		case websocket.StatusGoingAway:
			return err
		case -1:
			logger.WithError(err).Warn("error from echo handler")
		default:
			return err
		}
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return nil
		} else if errors.Is(err, context.Canceled) {
			return err
		} else if err != nil {
			logger.WithError(err).Warn("error from echo handler")
		}
	}
}

func Echo(ctx context.Context, c *websocket.Conn, l *rate.Limiter) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	err := l.Wait(ctx)
	if err != nil {
		return err
	}
	typ, r, err := c.Reader(ctx)
	if err != nil {
		return err
	}
	w, err := c.Writer(ctx, typ)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}
	return w.Close()
}
