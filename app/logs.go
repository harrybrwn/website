package app

import (
	"context"
	"database/sql"
	"net"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"harrybrown.com/pkg/db"
)

type RequestLog struct {
	ID          int           `json:"id"`
	Method      string        `json:"method"`
	Status      int           `json:"status"`
	IP          string        `json:"ip"`
	URI         string        `json:"uri"`
	Referer     string        `json:"referer"`
	UserAgent   string        `json:"user_agent"`
	Latency     time.Duration `json:"latency"`
	Error       error         `json:"error"`
	RequestedAt time.Time     `json:"requested_at"`
}

type LogManager struct {
	db     db.DB
	logger logrus.FieldLogger
}

const insertLogQuery = `
INSERT INTO request_log
	(method, status, ip, uri, referer, user_agent, latency, error)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8)`

func (lm *LogManager) Write(ctx context.Context, l *RequestLog) error {
	var errmsg string
	if l.Error != nil {
		errmsg = l.Error.Error()
	}
	var referer interface{}
	if len(l.Referer) != 0 {
		referer = l.Referer
	} else {
		referer = nil
	}
	_, err := lm.db.ExecContext(
		ctx,
		insertLogQuery,
		l.Method,
		l.Status,
		l.IP,
		l.URI,
		referer,
		l.UserAgent,
		l.Latency,
		errmsg,
	)
	return err
}

const getLogsQuery = `SELECT
		id,
		method,
		status,
		ip,
		uri,
		referer,
		user_agent,
		latency,
		error,
		requested_at
	FROM request_log
	WHERE id >= $1
	ORDER BY requested_at `

func (lm *LogManager) Get(ctx context.Context, limit, startID int, rev bool) ([]RequestLog, error) {
	var (
		res   = make([]RequestLog, 0, limit)
		query = getLogsQuery
	)
	if rev {
		query += "DESC"
	}
	rows, err := lm.db.QueryContext(ctx, query+" LIMIT $2", startID, limit)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var (
			l         RequestLog
			referrer  sql.NullString
			errString string
		)
		err = rows.Scan(
			&l.ID,
			&l.Method,
			&l.Status,
			&l.IP,
			&l.URI,
			&referrer,
			&l.UserAgent,
			&l.Latency,
			&errString,
			&l.RequestedAt,
		)
		if err != nil {
			rows.Close()
			return nil, err
		}
		l.Referer = referrer.String
		l.Error = errors.New(errString)
		res = append(res, l)
	}
	return res, rows.Close()
}

func LogRequest(logger logrus.FieldLogger, l *RequestLog) {
	fields := logrus.Fields{
		"method":     l.Method,
		"status":     l.Status,
		"ip":         l.IP,
		"uri":        l.URI,
		"referer":    l.Referer,
		"user_agent": l.UserAgent,
		"latency":    l.Latency,
		"latency_ms": float64(l.Latency) / 1.0e6,
	}
	if l.Error != nil {
		fields["error"] = l.Error
		logger.WithFields(fields).Error("request")
	} else {
		logger.WithFields(fields).Info("request")
	}
}

func RequestLogRecorder(db db.DB, logger logrus.FieldLogger) echo.MiddlewareFunc {
	logs := LogManager{db: db, logger: logger}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			res := c.Response()
			ctx := req.Context()
			ip := req.Header.Get("CF-Connecting-IP")
			if ip == "" {
				ip, _, _ = net.SplitHostPort(req.RemoteAddr)
			}
			err := next(c)
			if err != nil {
				c.Error(err)
			}
			l := RequestLog{
				Method:    req.Method,
				Status:    res.Status,
				IP:        ip,
				URI:       req.RequestURI,
				Referer:   req.Header.Get("Referer"),
				UserAgent: req.Header.Get("User-Agent"),
				Latency:   time.Since(start),
				Error:     err,
			}
			LogRequest(logger, &l)
			e := logs.Write(ctx, &l)
			if e != nil {
				logger.WithError(e).Error("could not record request")
			}
			if err != nil {
				return err
			}
			return e
		}
	}
}
