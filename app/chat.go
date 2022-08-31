package app

import (
	"context"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"harrybrown.com/app/chat"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/db"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/ws"
	"nhooyr.io/websocket"
)

func CreateChatRoom(store chat.Store) func(c echo.Context) error {
	return func(c echo.Context) error {
		claims := auth.GetClaims(c)
		if claims == nil {
			return echo.ErrUnauthorized.SetInternal(auth.ErrNoClaims)
		}
		query := c.Request().URL.Query()
		public, err := strconv.ParseBool(query.Get("public"))
		if err != nil {
			return echo.ErrBadRequest.SetInternal(err)
		}
		name := query.Get("name")
		logger.WithFields(logrus.Fields{
			"name":   name,
			"public": public,
		}).Info("creating room")
		room, err := store.CreateRoom(c.Request().Context(), claims.ID, name, public)
		if err != nil {
			return echo.ErrInternalServerError.SetInternal(err)
		}
		return c.JSON(200, room)
	}
}

func ChatRoomConnect(store chat.Store, rdb redis.UniversalClient) func(c echo.Context) error {
	return func(c echo.Context) error {
		var params struct {
			ID   int `param:"id"`
			User int `query:"user"`
		}
		if err := c.Bind(&params); err != nil {
			return err
		}
		claims := auth.GetClaims(c)
		if claims != nil {
			params.User = claims.ID
		}

		logger := logger.WithFields(logrus.Fields{
			"room_id": params.ID,
			"user_id": params.User,
		})

		// TODO check that ID is an existing room and that the requester has access to it.
		var (
			ctx  = log.StashInContext(c.Request().Context(), logger)
			room = chat.OpenRoom(store, rdb, params.ID, params.User)
		)
		err := room.Exists(ctx)
		if err != nil {
			return err
		}
		conn, err := ws.Accept(c.Response(), c.Request(), &ws.AcceptOptions{})
		if err != nil {
			return echo.ErrInternalServerError.SetInternal(err)
		}
		defer func() {
			conn.Close(websocket.StatusInternalError, "closing and returning from handler")
			logger.Info("stopping websocket")
		}()
		logger.Info("websocket connected")

		var (
			ps = chat.NewPubSub(rdb, params.ID, params.User)
			s  = chat.NewSocket(conn)
		)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if err = room.Start(ctx, ps, s); err != nil {
			return err
		}
		return nil
	}
}

func ListMessages(store chat.Store) echo.HandlerFunc {
	return func(c echo.Context) error {
		var p = struct {
			ID     int `param:"id"`
			Prev   int `query:"prev"`
			Offset int `query:"offset"`
			Limit  int `query:"limit"`
		}{
			Limit: 10,
		}
		err := c.Bind(&p)
		if err != nil {
			return err
		}
		ctx := c.Request().Context()
		msgs, err := store.Messages(ctx, p.ID, db.PaginationOpts{
			Prev:   p.Prev,
			Offset: p.Offset,
			Limit:  p.Limit,
		})
		if err != nil {
			return echo.ErrNotFound.SetInternal(err)
		}
		return c.JSON(200, map[string]interface{}{"messages": msgs})
	}
}

func GetRoom(store chat.Store) echo.HandlerFunc {
	type request struct {
		ID int `param:"id"`
	}
	return func(c echo.Context) error {
		var r request
		err := c.Bind(&r)
		if err != nil {
			return err
		}
		logger.WithField("id", r.ID).Info("got room request")
		room, err := store.GetRoom(c.Request().Context(), r.ID)
		if err != nil {
			logger.WithError(err).Error("failed to get room")
			return echo.ErrNotFound.SetInternal(err)
		}
		return c.JSON(200, room)
	}
}
