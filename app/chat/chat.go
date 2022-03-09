package chat

import (
	"context"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"harrybrown.com/pkg/db"
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
	ID        int       `json:"id"`
	OwnerID   int       `json:"owner_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
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
	const query = `INSERT INTO chatroom_messages (room, user_id, message) ` +
		`VALUES ($1, $2, $3)`
	_, err := rs.db.ExecContext(ctx, query, msg.Room, msg.UserID, msg.Body)
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
		switch websocket.CloseStatus(err) {
		case websocket.StatusNormalClosure:
			logger.Info("closing with status normal closure")
			return nil
		default:
			if err != nil {
				logger.WithError(err).Error("stopping socket")
				return err
			}
		}
	}
}

func EchoRead(ctx context.Context, c *websocket.Conn) {
}

func EchoWrite(ctx context.Context, c *websocket.Conn) {
}

func Echo(ctx context.Context, c *websocket.Conn, l *rate.Limiter) error {
	logger.Info("doing echo")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	err := l.Wait(ctx)
	if err != nil {
		return err
	}
	var (
		wg   sync.WaitGroup
		errs = make(chan error)
	)
	wg.Add(2)
	go func() {
		wg.Wait()
		close(errs)
	}()
	go func() {
		defer wg.Done()
		_, r, err := c.Reader(ctx)
		if err != nil {
			errs <- err
			return
		}
		io.Copy(os.Stdout, r)
	}()
	go func() {
		defer wg.Done()
		w, err := c.Writer(ctx, websocket.MessageText)
		if err != nil {
			errs <- err
			return
		}
		time.Sleep(time.Second * 5)
		w.Write([]byte("got it"))
		w.Close()
	}()
	err = <-errs
	return err
}
