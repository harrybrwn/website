package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oschwald/geoip2-golang"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/urifs"
	"harrybrown.com/pkg/web"
)

var logger = log.SetLogger(log.New(
	log.WithEnv(),
	log.WithFormat(log.JSONFormat),
	log.WithServiceName("geoip"),
))

func main() {
	var (
		start = time.Now()
		files []string
		port  = 8084
	)
	flag.StringArrayVarP(&files, "file", "f", files, "use a mmdb file")
	flag.IntVarP(&port, "port", "p", port, "port to run the server on")
	flag.Parse()
	uris, err := parseURIs(files)
	if err != nil {
		logger.Fatal(errors.Wrap(err, "failed to parse uri flags"))
	}

	db, err := FetchDatabases(uris...)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()
	logger.WithFields(log.Fields{
		"city-db":    db.City != nil,
		"asn-db":     db.ASN != nil,
		"country-db": db.Country != nil,
		"duration":   time.Since(start),
	}).Info("loaded GeoIP database")

	r := chi.NewRouter()
	r.Use(web.AccessLog(logger))
	r.Use(web.Metrics())
	r.Get("/{ip}", db.Info)
	r.Get("/", EchoIP)
	r.Get("/favicon.ico", send404)
	r.With(web.PrivateOnly(logger)).Handle("/metrics", web.MetricsHandler())
	r.Head("/health/ready", func(http.ResponseWriter, *http.Request) {})
	err = web.ListenAndServe(fmt.Sprintf(":%d", port), r)
	if err != nil {
		logger.WithError(err).Fatal("listen and serve failed")
	}
}

type GeoData struct {
	ASN     *geoip2.Reader
	City    *geoip2.Reader
	Country *geoip2.Reader
}

func (gd *GeoData) Info(w http.ResponseWriter, r *http.Request) {
	ipa := chi.URLParam(r, "ip")
	if len(ipa) == 0 {
		web.WriteError(w, web.StatusError(http.StatusBadRequest, errors.New("no ip address")))
		return
	}
	var (
		ip  net.IP
		err error
	)
	if ipa == "self" {
		ip, err = getIP(r)
		if err != nil {
			web.WriteError(w, err)
			return
		}
	} else {
		ip = net.ParseIP(ipa)
		if ip == nil {
			err := fmt.Errorf("invalid ip address %q", ipa)
			web.WriteError(w, web.StatusError(http.StatusBadRequest, err, err.Error()))
			return
		}
	}

	res, err := gd.City.City(ip)
	if err != nil {
		web.WriteError(w, err)
		return
	}
	asn, err := gd.ASN.ASN(ip)
	if err != nil {
		web.WriteError(w, err)
		return
	}
	writeJSON(w, map[string]any{
		"ip":          ip,
		"city":        res.City,
		"country":     res.Country,
		"location":    res.Location,
		"postal_code": res.Postal.Code,
		"traits":      res.Traits,
		"asn":         asn,
	})
}

func EchoIP(w http.ResponseWriter, r *http.Request) {
	ip, err := getIP(r)
	if err != nil {
		web.WriteError(w, err)
		return
	}
	accept := r.Header.Get("Accept")
	switch accept {
	default:
		fallthrough
	case "text/plain":
		w.Write([]byte(ip.String()))
	case "application/json":
		writeJSON(w, map[string]any{"ip": ip.String()})
	}
}

func send404(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }

func (gd *GeoData) Close() error {
	for _, closer := range []io.Closer{gd.ASN, gd.City, gd.Country} {
		if closer != nil {
			if err := closer.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func getIP(r *http.Request) (net.IP, error) {
	h := firstHeader(r.Header, "Cf-Connecting-Ip", "X-Forwarded-For", "X-Real-IP")
	if len(h) == 0 {
		var err error
		h, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return nil, web.StatusError(http.StatusBadRequest, err)
		}
	}
	ips := parseIPList(h)
	if len(ips) == 0 {
		return nil, errors.New("no ip address found")
	}
	return ips[0], nil
}

func parseIPList(raw string) []net.IP {
	parts := strings.Split(raw, ",")
	ips := make([]net.IP, 0, len(parts))
	for _, p := range parts {
		ip := net.ParseIP(strings.Trim(p, " \n\r\t"))
		if ip == nil {
			continue
		}
		ips = append(ips, ip)
	}
	return ips
}

func firstHeader(h http.Header, keys ...string) (v string) {
	for _, k := range keys {
		v = h.Get(k)
		if len(v) > 0 {
			break
		}
	}
	return v
}

func FetchDatabases(uris ...*url.URL) (*GeoData, error) {
	var g GeoData
	for _, uri := range uris {
		var r *geoip2.Reader
		// reader, err := openURI(uri)
		reader, err := urifs.Open(uri)
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(reader)
		if err != nil {
			reader.Close()
			return nil, err
		}
		if err = reader.Close(); err != nil {
			return nil, err
		}
		r, err = geoip2.FromBytes(b)
		if err != nil {
			return nil, err
		}

		if r == nil {
			return nil, errors.New("unknown file scheme")
		}
		switch r.Metadata().DatabaseType {
		case "GeoLite2-ASN":
			g.ASN = r
		case "GeoLite2-City":
			g.City = r
		case "GeoLite2-Country":
			g.Country = r
		default:
			return nil, errors.New("unknown mmdb database type")
		}
	}
	if g.City == nil {
		return nil, errors.New("must have at least the city database")
	}
	return &g, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		logger.WithError(err).Error("failed to marshal json")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func parseURIs(s []string) ([]*url.URL, error) {
	var (
		err  error
		uris = make([]*url.URL, len(s))
	)
	for i, f := range s {
		uris[i], err = url.Parse(f)
		if err != nil {
			return nil, err
		}
	}
	return uris, nil
}
