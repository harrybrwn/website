package web

import (
	"net"
	"net/http"
	"time"

	flag "github.com/spf13/pflag"
	"harrybrown.com/pkg/log"
)

var (
	logger                 = log.GetLogger()
	SSLCertificateFileFlag string
	SSLKeyFileFlag         string
)

func init() {
	flag.StringVar(&SSLCertificateFileFlag, "cert", SSLCertificateFileFlag, "ssl certificate file")
	flag.StringVar(&SSLKeyFileFlag, "key", SSLKeyFileFlag, "ssl key file")
}

func ListenAndServe(addr string, h http.Handler) error {
	if SSLCertificateFileFlag == "" && SSLKeyFileFlag == "" {
		logger.WithFields(log.Fields{
			"address": addr,
			"time":    time.Now(),
		}).Info("starting server")
		return http.ListenAndServe(addr, h)
	}
	logger.WithFields(log.Fields{
		"address":  addr,
		"ssl-cert": SSLCertificateFileFlag,
		"ssl-key":  SSLKeyFileFlag,
		"time":     time.Now(),
	}).Info("starting server with tls")
	return http.ListenAndServeTLS(addr, SSLCertificateFileFlag, SSLKeyFileFlag, h)
}

func PrivateOnly(logger *log.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, err := getIP(r)
			if err != nil {
				WriteError(w, StatusError(http.StatusBadRequest, err, "bad request ip address"))
				return
			}
			if !ip.IsPrivate() {
				w.WriteHeader(http.StatusForbidden)
				logger.WithField("ip", ip.String()).Info("skipping non-private ip address")
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}

func getIP(r *http.Request) (net.IP, error) {
	var (
		err error
		val string
		ip  net.IP
	)
	for _, key := range []string{
		"Cf-Connecting-Ip",
		"X-Forwarded-For",
		"X-Real-IP",
	} {
		val = r.Header.Get(key)
		if len(val) == 0 {
			continue
		}
		ip = net.ParseIP(val)
		if ip == nil {
			continue
		}
		goto found
	}
	val, _, err = net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, StatusError(http.StatusBadRequest, err)
	}
	ip = net.ParseIP(val)
found:
	return ip, nil
}
