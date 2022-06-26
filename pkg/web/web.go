package web

import (
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
