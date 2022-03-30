package ws

import (
	"context"
	"io"
	"net/http"

	"nhooyr.io/websocket"
)

type (
	AcceptOptions = websocket.AcceptOptions
	DialOptions   = websocket.DialOptions
	MessageType   = websocket.MessageType
	StatusCode    = websocket.StatusCode
	CloseError    = websocket.CloseError
)

const (
	StatusNormalClosure           StatusCode = websocket.StatusNormalClosure
	StatusGoingAway               StatusCode = websocket.StatusGoingAway
	StatusProtocolError           StatusCode = websocket.StatusProtocolError
	StatusUnsupportedData         StatusCode = websocket.StatusUnsupportedData
	StatusNoStatusRcvd            StatusCode = websocket.StatusNoStatusRcvd
	StatusAbnormalClosure         StatusCode = websocket.StatusAbnormalClosure
	StatusInvalidFramePayloadData StatusCode = websocket.StatusInvalidFramePayloadData
	StatusPolicyViolation         StatusCode = websocket.StatusPolicyViolation
	StatusMessageTooBig           StatusCode = websocket.StatusMessageTooBig
	StatusMandatoryExtension      StatusCode = websocket.StatusMandatoryExtension
	StatusInternalError           StatusCode = websocket.StatusInternalError
	StatusServiceRestart          StatusCode = websocket.StatusServiceRestart
	StatusTryAgainLater           StatusCode = websocket.StatusTryAgainLater
	StatusBadGateway              StatusCode = websocket.StatusBadGateway
	StatusTLSHandshake            StatusCode = websocket.StatusTLSHandshake

	MessageBinary MessageType = websocket.MessageBinary
	MessageText   MessageType = websocket.MessageText
)

type Connection interface {
	Reader(ctx context.Context) (MessageType, io.Reader, error)
	Writer(ctx context.Context, typ MessageType) (io.WriteCloser, error)
	Close(code StatusCode, reason string) error
	Ping(ctx context.Context) error
	Subprotocol() string
}

func Accept(w http.ResponseWriter, r *http.Request, opts *AcceptOptions) (Connection, error) {
	return websocket.Accept(w, r, opts)
}

func Dial(ctx context.Context, u string, opts *DialOptions) (Connection, *http.Response, error) {
	return websocket.Dial(ctx, u, opts)
}

func CloseStatus(err error) StatusCode {
	return websocket.CloseStatus(err)
}
