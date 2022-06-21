package web

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"harrybrown.com/pkg/log"
)

func AccessLog(logger *log.Logger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			resp := logResponse{ResponseWriter: w}
			h.ServeHTTP(&resp, r)
			switch r.RequestURI {
			case "/metrics":
				return
			}
			s := resp.status
			logger := logger.WithFields(logrus.Fields{
				"host":        r.Host,
				"method":      r.Method,
				"uri":         r.RequestURI,
				"status":      s,
				"query":       r.URL.RawQuery,
				"remote_addr": r.RemoteAddr,
				"duration":    time.Since(start).String(),
			})
			if s < 400 {
				logger.Info("request handled")
			} else if s < 500 {
				logger.Warn("request handled")
			} else if s >= 500 {
				logger.Error("request handled")
			}
		})
	}
}

type logResponse struct {
	http.ResponseWriter
	status int
}

func (r *logResponse) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *logResponse) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = 200
	}
	return r.ResponseWriter.Write(b)
}
