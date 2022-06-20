package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/oschwald/geoip2-golang"
	flag "github.com/spf13/pflag"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

var logger = log.SetLogger(log.New(log.WithEnv(), log.WithFormat(log.JSONFormat)))

func main() {
	var (
		start = time.Now()
		files []string
	)
	flag.StringArrayVarP(&files, "file", "f", files, "use a mmdb file")
	flag.Parse()
	uris, err := parseURIs(files)
	if err != nil {
		logger.Fatal(err)
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
	r.Get("/{ip}", db.Info)
	r.Get("/", EchoIP)
	r.Get("/favicon.ico", send404)
	addr := ":8084"
	logger.WithField("address", addr).Info("starting server")
	http.ListenAndServe(addr, r)
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
	}

	if ip == nil {
		err := fmt.Errorf("invalid ip address %q", ipa)
		web.WriteError(w, web.StatusError(http.StatusBadRequest, err, err.Error()))
		return
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
	h := firstHeader(r.Header, "X-Forwarded-For", "X-Real-IP")
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
		reader, err := openURI(uri)
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

func openURI(uri *url.URL) (io.ReadCloser, error) {
	switch uri.Scheme {
	case "file":
		p := uri.Path
		if uri.Host != "" {
			p = filepath.Join("./", uri.Host, uri.Path)
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		return f, nil
	case "s3":
		session, err := s3SessionFromURI(uri, getenv("AWS_S3_REGION", "us-east-1"))
		if err != nil {
			return nil, err
		}
		in := objectRequestFromURI(uri)
		if in == nil {
			return nil, errors.New("invalid s3 uri")
		}
		resp, err := s3.New(session).GetObject(in)
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	case "http", "https":
		resp, err := http.DefaultClient.Do(&http.Request{
			URL:    uri,
			Method: "GET",
			Host:   uri.Host,
			Close:  true,
		})
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	default:
		return nil, errors.New("unknown file scheme")
	}
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

func s3SessionFromURI(uri *url.URL, region string) (client.ConfigProvider, error) {
	pw, ok := uri.User.Password()
	if !ok {
		return nil, errors.New("no s3 secret key")
	}
	var endpoint *string
	if uri.Host != "" {
		endpoint = aws.String(uri.Host)
	}
	return session.NewSession(&aws.Config{
		Endpoint:         endpoint,
		Region:           aws.String(region),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
		Logger:           aws.LoggerFunc(logger.WithField("client", "s3").Info),
		Credentials:      credentials.NewStaticCredentials(uri.User.Username(), pw, ""),
	})
}

func objectRequestFromURI(uri *url.URL) *s3.GetObjectInput {
	parts := strings.Split(uri.Path, string(filepath.Separator))
	if len(parts) == 0 {
		return nil
	} else if parts[0] == "" {
		parts = parts[1:]
	}
	if len(parts) < 2 {
		return nil
	}
	return &s3.GetObjectInput{
		Bucket: aws.String(parts[0]),
		Key:    aws.String(filepath.Join(parts[1:]...)),
	}
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

func getenv(key, defaultValue string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return val
}
