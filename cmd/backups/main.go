package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-chi/chi/v5"
	flag "github.com/spf13/pflag"
	"gopkg.hrry.dev/homelab/pkg/db"
	"gopkg.hrry.dev/homelab/pkg/log"
	"gopkg.hrry.dev/homelab/pkg/web"
)

var logger = log.New(
	log.WithFormat(log.TextFormat),
	log.WithLevel(log.DebugLevel),
	log.WithServiceName("backups"),
)

func init() { log.SetLogger(logger) }

func main() {
	var (
		port = 8082
	)
	flag.IntVarP(&port, "port", "p", port, "port to run the server on")
	flag.Parse()

	awsSession, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(s3Endpoint()),
		Region:           aws.String("us-west-0"),
		Credentials:      s3Credentials(),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
		Logger:           aws.LoggerFunc(logger.WithField("client", "s3").Info),
	})
	if err != nil {
		logger.Fatalf("%+v", err)
	}
	s3 := s3.New(awsSession)
	config := newConfig(time.Hour*12, 6, getenv("BACKUPS_BUCKET", "db-backups"))
	info := &JobInfo{config: config, db: GetDBInfo()}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go Start(ctx, s3, info)

	r := chi.NewRouter()
	r.Use(web.AccessLog(logger))
	r.Use(web.Metrics())
	r.Put("/config", handleUpdateConfig(config))
	r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		err := writeJSON(w, config)
		if err != nil {
			web.WriteError(w, err)
			return
		}
	})
	r.Post("/backup/postgres", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		backup, err := BackupPostgres(r.Context(), info.db, s3, config.Bucket)
		if err != nil {
			web.WriteError(w, web.WrapError(err, "failed backup"))
			return
		}
		w.WriteHeader(200)
		writeJSON(w, map[string]any{"status": "success", "file": backup.Filename})
	})
	r.Handle("/metrics", web.MetricsHandler())
	r.Head("/health/ready", func(w http.ResponseWriter, r *http.Request) {})

	addr := fmt.Sprintf(":%d", port)
	logger.WithField("addr", addr).Info("starting http")
	http.ListenAndServe(addr, r)
}

func handleUpdateConfig(config *JobConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		var (
			req UpdateConfigRequest
			res UpdateConfigResponse
		)
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logger.WithError(err).Debug("json decode failure")
			w.WriteHeader(http.StatusBadRequest)
			res.Message = "invalid json"
			writeJSON(w, &res)
			return
		}
		err = config.Update(&req)
		if err != nil {
			logger.WithError(err).Debug("failed to update config")
			w.WriteHeader(http.StatusBadRequest)
			res.Message = "invalid configuration"
			writeJSON(w, &res)
			return
		}
		w.WriteHeader(200)
		res.Message = "success"
		writeJSON(w, &res)
	}
}

func s3Endpoint() string {
	endpoint := os.Getenv("S3_ENDPOINT")
	if len(endpoint) > 0 {
		return endpoint
	}
	return "localhost:9000"
}

func s3Credentials() *credentials.Credentials {
	credsProvider, err := db.S3CredentialsProvider()
	if err != nil {
		logger.Error(err)
		return nil
	}
	return credentials.NewCredentials(credsProvider)
}

func writeJSON(w io.Writer, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		logger.WithError(err).Error("failed to marshal json")
		return err
	}
	_, err = w.Write(b)
	if err != nil {
		logger.WithError(err).Error("failed to write json data")
		return err
	}
	return nil
}

func getenv(key, defaultValue string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return val
}
