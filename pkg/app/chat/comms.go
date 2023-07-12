package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"gopkg.hrry.dev/homelab/pkg/ws"
	"nhooyr.io/websocket"
)

var ErrEmptyBody = errors.New("empty message body")

type PubSub interface {
	// Publish a message on the message queue
	Pub(context.Context, *Message) error
	// Subscribe to messages being published by other connections
	Sub(context.Context) <-chan Message
}

func NewPubSub(rd redis.UniversalClient, room, user int) *pubsub {
	return &pubsub{
		RDB:  rd,
		Room: room,
		User: user,
	}
}

type pubsub struct {
	RDB        redis.UniversalClient
	Room, User int
}

// Pub publishes a message to the channel
func (c *pubsub) Pub(ctx context.Context, msg *Message) error {
	key := fmt.Sprintf("room:%d:user:%d", c.Room, c.User)
	msg.Room = c.Room
	msg.UserID = c.User
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.RDB.Publish(ctx, key, raw).Err()
}

// Sub subscribes to a channel to listen to new messages
func (c *pubsub) Sub(ctx context.Context) <-chan Message {
	msgs := make(chan Message)
	pubsub := c.RDB.PSubscribe(ctx, fmt.Sprintf("room:%d:user:*", c.Room))
	ch := pubsub.Channel()
	go func() {
		defer func() {
			if err := pubsub.Close(); err != nil {
				logger.WithError(err).Warn("failed to close redis PubSub")
			}
			close(msgs)
		}()
		for message := range ch {
			var msg Message
			err := json.Unmarshal([]byte(message.Payload), &msg)
			if err != nil {
				logger.WithError(err).Error("failed to unmarshal chat message from pubsub")
				continue
			}
			if msg.UserID == c.User {
				// Ignore messages from ourselfs
				continue
			}
			select {
			case msgs <- msg:
			case <-ctx.Done():
				logger.WithError(ctx.Err()).Warn("stopping chat subscription")
				return
			}
		}
	}()
	return msgs
}

type Socket interface {
	// Send a message down the connection
	Send(context.Context, *Message) error
	// Listen for new messages from the connection
	Recv(context.Context) (*Message, error)
}

func NewSocket(conn ws.Connection) Socket {
	return &socket{conn: conn}
}

type socket struct {
	conn ws.Connection
}

func (s *socket) Send(ctx context.Context, msg *Message) error {
	if msg == nil {
		return errors.New("cannot send nil message")
	}
	w, err := s.conn.Writer(ctx, websocket.MessageText)
	if err != nil {
		return errors.Wrap(err, "failed to get writer")
	}
	err = json.NewEncoder(w).Encode(msg)
	if err != nil {
		w.Close()
		return err
	}
	if err = w.Close(); err != nil {
		return errors.Wrap(err, "could not close websocket writer")
	}
	return nil
}

func (s *socket) Recv(ctx context.Context) (*Message, error) {
	_, r, err := s.conn.Reader(ctx)
	if err != nil {
		return nil, err
	}
	var msg Message
	err = json.NewDecoder(r).Decode(&msg)
	if err != nil {
		return nil, err
	}
	if len(msg.Body) == 0 {
		return nil, ErrEmptyBody
	}
	msg.CreatedAt = time.Now()
	return &msg, nil
}
