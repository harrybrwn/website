package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sirupsen/logrus"
)

type Config struct {
	AccountID  int
	EditionIDs []string
	LicenseKey string
}

type clientOptions struct {
	logger *logrus.Logger
	env    string
	mode   Mode
}

type ClientOpt func(*clientOptions)

func WithEnv(e string) ClientOpt { return func(co *clientOptions) { co.env = e } }
func WithMode(m Mode) ClientOpt  { return func(co *clientOptions) { co.mode = m } }

func NewClient(config *Config, awscfg *aws.Config, opts ...ClientOpt) *client {
	var options = clientOptions{
		logger: logrus.StandardLogger(),
		env:    "dev",
	}
	for _, o := range opts {
		o(&options)
	}
	return &client{
		accountID:  config.AccountID,
		licenseKey: config.LicenseKey,
		client:     http.DefaultClient,
		s3: s3.NewFromConfig(*awscfg, func(o *s3.Options) {
			o.UsePathStyle = true
			if Debug {
				o.ClientLogMode = aws.LogRequest | aws.LogResponse
			}
			o.UseARNRegion = false
		}),
		clientOptions: options,
	}
}

type client struct {
	accountID  int
	licenseKey string
	client     *http.Client
	s3         *s3.Client
	clientOptions
}

func (c *client) Download(ctx context.Context, edition string) (r *response, err error) {
	logger := c.logger.WithField("edition_id", edition)
	switch strings.ToLower(edition) {
	case "geolite2-asn", "geolite2-city", "geolite2-country":
		logger.Info("downloading database")
		r, err = c.database(ctx, edition)
		if err != nil {
			return nil, err
		}
		r.Type = TypeMMDB
	case "geolite2-asn-csv", "geolite2-city-csv", "geolite2-country-csv":
		logger.Info("downloading csv")
		r, err = c.csv(ctx, edition, "zip")
		if err != nil {
			return nil, err
		}
		r.Type = TypeCSV
	default:
		return nil, errors.New("unknown edition ID")
	}
	return r, err
}

type file struct {
	io.ReadCloser
	name string
}

type FileType uint8

const (
	TypeCSV FileType = iota
	TypeMMDB
)

type response struct {
	Files    []file
	Modified time.Time
	Hash     string
	Type     FileType
}

func (r *response) Close() error {
	return closeAll(r.Files)
}

func (c *client) date() string {
	return time.Now().Format(time.DateOnly)
}

func (c *client) write(ctx context.Context, base string, file *file) error {
	date := time.Now().Format(time.DateOnly)
	dir := filepath.Join(base, date, c.env)
	path := filepath.Join(dir, file.name)
	_ = os.MkdirAll(dir, 0755)
	c.logger.Println("writing to", path)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, file)
	return err
}

func (c *client) upload(ctx context.Context, base string, file *file) error {
	date := time.Now().Format(time.DateOnly)
	key := filepath.Join(c.env, date, file.name)
	logger := c.logger.WithFields(logrus.Fields{
		"bucket": base,
		"key":    key,
	})
	err := c.deleteLatest(ctx, base)
	if err != nil {
		logger.WithError(err).Warn("failed to delete 'latest' keys")
	}
	defer func() {
		err := c.copyToLatest(ctx, base, file)
		if err != nil {
			logger.WithError(err).Warn("failed to copy download to the 'latest' folder")
		}
	}()
	logger.Info("uploading")
	_, err = c.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &base,
		Key:    &key,
	})
	if err == nil {
		return os.ErrExist
	}
	b, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	body := bytes.NewReader(b)
	putReq := &s3.PutObjectInput{
		Bucket: &base,
		Key:    &key,
		Body:   body,
	}
	_, err = c.s3.PutObject(ctx, putReq)
	if err != nil {
		return err
	}
	logger.Println(key, "upload finished")
	return nil
}

func (c *client) copyToLatest(ctx context.Context, base string, file *file) error {
	latestKey := filepath.Join(c.env, "latest", file.name)
	copySource := filepath.Join(base, c.env, c.date(), file.name)
	c.logger.Infof("copying %q to %q", copySource, latestKey)
	_, err := c.s3.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     &base,
		CopySource: &copySource,
		Key:        &latestKey,
	})
	return err
}

func (c *client) deleteLatest(ctx context.Context, base string) error {
	prefix := filepath.Join(c.env, "latest/")
	keys := make([]types.ObjectIdentifier, 0)
	objs, err := c.s3.ListObjects(ctx, &s3.ListObjectsInput{
		Bucket: &base,
		Prefix: &prefix,
	})
	if err != nil {
		return err
	}
	for _, o := range objs.Contents {
		keys = append(keys, types.ObjectIdentifier{Key: o.Key})
	}
	_, err = c.s3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: &base,
		Delete: &types.Delete{
			Objects: keys,
		},
	})
	return err
}

func (c *client) database(ctx context.Context, edition string) (*response, error) {
	req := http.Request{
		URL: &url.URL{
			Scheme: "https",
			Host:   "updates.maxmind.com",
			Path: path.Join(
				"/geoip/databases",
				edition,
				"update",
			),
			RawQuery: (&url.Values{
				"db_md5": {"00000000000000000000000000000000"}, // TODO
			}).Encode(),
		},
		Method: "GET",
		Header: http.Header{
			"User-Agent": {"geoipupdate/0.1"},
		},
	}
	req.SetBasicAuth(strconv.Itoa(c.accountID), c.licenseKey)
	return c.invoke(req.WithContext(ctx))
}

func (c *client) csv(ctx context.Context, edition, suffix string) (*response, error) {
	req := http.Request{
		URL: &url.URL{
			Scheme: "https",
			Host:   "updates.maxmind.com",
			Path:   "/app/geoip_download",
			RawQuery: (&url.Values{
				"edition_id":  {edition},
				"license_key": {c.licenseKey},
				"suffix":      {suffix},
			}).Encode(),
		},
	}
	return c.invoke(req.WithContext(ctx))
}

func (c *client) invoke(req *http.Request) (*response, error) {
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == 404 {
		return nil, errors.New(res.Status)
	}
	mod, err := time.Parse(time.RFC1123, res.Header.Get("Last-Modified"))
	if err != nil {
		// a bit heavy handed, might want to throw out a warning here instead
		return nil, err
	}
	var r = response{Modified: mod, Hash: res.Header.Get("X-Database-MD5")}
	ct := res.Header.Get("Content-Type")
	filename, err := filenameFromDisposition(res)
	if err != nil {
		return nil, err
	}
	c.logger.WithFields(logrus.Fields{
		"status":               res.StatusCode,
		"proto":                res.Proto,
		"content-length":       res.ContentLength,
		"content-type":         ct,
		"last-modified":        mod,
		"db-hash":              r.Hash,
		"disposition-filename": filename,
		"headers":              res.Header,
	}).Debug("download result")

	switch {
	case strings.HasPrefix(ct, "application/gzip"):
		gzr, err := gzip.NewReader(res.Body)
		if err != nil {
			return nil, err
		}
		if filepath.Ext(filename) == ".gz" {
			// We are decompressing it automatically, no need to give it a gzip
			// extension.
			filename = strings.TrimSuffix(filename, ".gz")
		}
		r.Files = []file{{ReadCloser: gzr, name: filename}}
	case strings.HasPrefix(ct, "application/zip"):
		defer res.Body.Close() // we will be reading the whole thing in memory
		var buf bytes.Buffer
		n, err := buf.ReadFrom(res.Body)
		if err != nil {
			return nil, err
		}
		zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), n)
		if err != nil {
			return nil, err
		}
		r.Files = make([]file, 0, len(zr.File))
		for _, f := range zr.File {
			rc, err := f.Open()
			if err != nil {
				// we might have stored open buffers previously so we need to
				// close them all out before we error
				closeAll(r.Files)
				return nil, err
			}
			_, name := filepath.Split(f.Name)
			r.Files = append(r.Files, file{ReadCloser: rc, name: name})
		}
	default:
		r.Files = []file{{ReadCloser: res.Body, name: filename}}
	}
	return &r, nil
}

func filenameFromDisposition(res *http.Response) (string, error) {
	disposition, params, err := mime.ParseMediaType(res.Header.Get("Content-Disposition"))
	if err != nil {
		return "", fmt.Errorf("failed to parse content-disposition header: %w", err)
	}
	filename, ok := params["filename"]
	if !ok || strings.ToLower(disposition) != "attachment" {
		return "", errors.New("failed to find filename in response")
	}
	_, name := filepath.Split(filename) // make sure there's no base directory
	return name, nil
}

func closeAll(files []file) (err error) {
	// TODO create an errorChain error
	for _, f := range files {
		if f.ReadCloser == nil {
			continue
		}
		e := f.Close()
		if err == nil && e != nil {
			err = e
		}
	}
	return err
}
