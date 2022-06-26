package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/loki/pkg/logproto"
	"harrybrown.com/pkg/log"
)

const (
	minioService     = "minio"
	minioLoggerLabel = "minio-logger-webhook"
	minioAuditLabel  = "minio-audit-webhook"
)

type MinioEntry interface {
	GetTime() time.Time
	GetDeploymentID() string
	JobLabel() string
}

func minioLoggingHookHandler[T MinioEntry](pusher logproto.PusherClient) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			buf   bytes.Buffer
			entry T
		)
		logger := logger.WithFields(log.Fields{
			"request_uri":         r.RequestURI,
			"content_length":      r.Header.Get("Content-Length"),
			"user_agent":          r.Header.Get("User-Agent"),
			"minio_deployment_id": r.Header.Get("X-Minio-Deployment-Id"),
			"minio_version":       r.Header.Get("X-Minio-Version"),
		})
		err := json.NewDecoder(io.TeeReader(r.Body, &buf)).Decode(&entry)
		if err != nil {
			logger.WithError(err).Error("could not read hook body")
			w.WriteHeader(500)
			return
		}
		if err = r.Body.Close(); err != nil {
			logger.WithError(err).Error("could not close request body")
			w.WriteHeader(500)
			return
		}
		if entry.GetDeploymentID() == "" {
			logger.Info("empty log line")
			w.WriteHeader(http.StatusAccepted)
			return
		}

		labels := fmt.Sprintf(
			`{service=%q,job=%q,type="logs"}`,
			minioService,
			entry.JobLabel(),
		)
		// TODO buffer these requests so they don't cause too many circular
		// writes to s3.
		_, err = pusher.Push(r.Context(), &logproto.PushRequest{
			Streams: []logproto.Stream{
				{
					Labels: labels,
					Entries: []logproto.Entry{
						{Timestamp: entry.GetTime(), Line: buf.String()},
					},
				},
			},
		})
		if err != nil {
			logger.WithError(err).Error("failed to push logs to loki")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.WithField("labels", labels).Info("sent log line to loki")
		w.WriteHeader(http.StatusAccepted)
	}
}

type ObjectVersion struct {
	ObjectName string `json:"objectName"`
	VersionID  string `json:"versionId,omitempty"`
}

type MinioLogEntry struct {
	DeploymentID string         `json:"deploymentid,omitempty"`
	Level        string         `json:"level"`
	LogKind      string         `json:"errKind"`
	Time         time.Time      `json:"time"`
	API          *minioLogAPI   `json:"api,omitempty"`
	RemoteHost   string         `json:"remotehost,omitempty"`
	Host         string         `json:"host,omitempty"`
	RequestID    string         `json:"requestID,omitempty"`
	UserAgent    string         `json:"userAgent,omitempty"`
	Message      string         `json:"message,omitempty"`
	Trace        *minioLogTrace `json:"error,omitempty"`
}

func (le *MinioLogEntry) GetTime() time.Time      { return le.Time }
func (le *MinioLogEntry) GetDeploymentID() string { return le.DeploymentID }
func (le *MinioLogEntry) JobLabel() string        { return minioLoggerLabel }

type minioLogAPI struct {
	Name string        `json:"name,omitempty"`
	Args *minioLogArgs `json:"args,omitempty"`
}

type minioLogArgs struct {
	Bucket    string            `json:"bucket,omitempty"`
	Object    string            `json:"object,omitempty"`
	VersionID string            `json:"versionId,omitempty"`
	Objects   []ObjectVersion   `json:"objects,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type minioLogTrace struct {
	Message   string                 `json:"message,omitempty"`
	Source    []string               `json:"source,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type MinioAuditEntry struct {
	Version      string    `json:"version"`
	DeploymentID string    `json:"deploymentid,omitempty"`
	Time         time.Time `json:"time"`
	Trigger      string    `json:"trigger"`
	API          struct {
		Name            string          `json:"name,omitempty"`
		Bucket          string          `json:"bucket,omitempty"`
		Object          string          `json:"object,omitempty"`
		Objects         []ObjectVersion `json:"objects,omitempty"`
		Status          string          `json:"status,omitempty"`
		StatusCode      int             `json:"statusCode,omitempty"`
		InputBytes      int64           `json:"rx"`
		OutputBytes     int64           `json:"tx"`
		TimeToFirstByte string          `json:"timeToFirstByte,omitempty"`
		TimeToResponse  string          `json:"timeToResponse,omitempty"`
	} `json:"api"`
	RemoteHost string                 `json:"remotehost,omitempty"`
	RequestID  string                 `json:"requestID,omitempty"`
	UserAgent  string                 `json:"userAgent,omitempty"`
	ReqClaims  map[string]interface{} `json:"requestClaims,omitempty"`
	ReqQuery   map[string]string      `json:"requestQuery,omitempty"`
	ReqHeader  headers                `json:"requestHeader,omitempty"`
	RespHeader headers                `json:"responseHeader,omitempty"`
	Tags       map[string]interface{} `json:"tags,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

func (ae *MinioAuditEntry) GetTime() time.Time      { return ae.Time }
func (ae *MinioAuditEntry) GetDeploymentID() string { return ae.DeploymentID }
func (ae *MinioAuditEntry) JobLabel() string        { return minioAuditLabel }

type headers map[string]string

func (h headers) get(key string) string {
	key = strings.ToLower(key)
	for k, v := range h {
		if strings.ToLower(k) == key {
			return v
		}
	}
	return ""
}
