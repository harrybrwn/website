package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"harrybrown.com/pkg/log"
)

func AccessLog(logger *log.Logger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			resp := logResponse{ResponseWriter: w}
			l := logger.WithFields(log.Fields{
				"host":        r.Host,
				"method":      r.Method,
				"uri":         r.RequestURI,
				"remote_addr": r.RemoteAddr,
				"query":       r.URL.RawQuery,
			})
			ctx := log.StashInContext(r.Context(), l)
			h.ServeHTTP(&resp, r.WithContext(ctx))
			switch r.RequestURI {
			case
				"/metrics",
				"/api/health/ready",
				"/api/health/alive":
				return
			}
			logAccess(l, resp.status, start, r)
		})
	}
}

func logAccess(logger log.FieldLogger, status int, start time.Time, r *http.Request) {
	d := time.Since(start)
	l := logger.WithFields(log.Fields{
		"status":      status,
		"duration_ms": d.Milliseconds(),
	})
	if status < 400 {
		l.Info("request handled")
	} else if status < 500 {
		l.Warn("request handled")
	} else if status >= 500 {
		l.Error("request handled")
	}
}

func Metrics() func(h http.Handler) http.Handler {
	requestCnt := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Number of http requests served.",
		},
		[]string{"code", "method", "uri"},
	)
	durationHst := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_response_time_seconds",
			Help: "Duration of HTTP requests.",
		},
		[]string{"uri"},
	)
	prometheus.MustRegister(
		requestCnt,
		durationHst,
	)
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			timer := prometheus.NewTimer(durationHst.WithLabelValues(r.RequestURI))
			resp := logResponse{ResponseWriter: w}
			h.ServeHTTP(&resp, r)
			timer.ObserveDuration()
			requestCnt.With(prometheus.Labels{
				"code":   strconv.FormatInt(int64(resp.status), 10),
				"method": r.Method,
				"uri":    r.RequestURI,
			}).Inc()
		})
	}
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
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
