package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-github/v43/github"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
	"google.golang.org/grpc"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/session"
	"harrybrown.com/pkg/web"
)

var (
	//go:embed index.html
	index []byte
	//go:embed login.html
	loginHTML []byte

	logger = log.SetLogger(log.New(
		log.WithEnv(),
		log.WithFormat(log.JSONFormat),
		log.WithServiceName("hooks"),
	))
)

func main() {
	var (
		host     string
		port     = 8889
		lokiAddr = "loki:9096"
	)
	if err := godotenv.Load(); err != nil {
		logger.WithError(err).Warn("could not load .env")
	}
	if err := validateEnv(); err != nil {
		logger.WithError(err).Fatal("invalid configuration")
	}

	flag.IntVar(&port, "port", port, "specify server port")
	flag.StringVar(&host, "host", os.Getenv("GH_HOOK_CALLBACK_HOST"), "server's domain name, used for creating webhooks")
	flag.Parse()

	logger.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339})
	if host == "" {
		logger.Fatal("no server host given, set -host or $SERVER_HOST")
	}

	gh := GithubAuthService{
		Sessions:    session.NewMemStore[ghSession](time.Minute),
		AuthSession: session.NewMemStore[oauthSession](time.Minute * 2),
		Config: oauth2.Config{
			ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			Endpoint:     githuboauth.Endpoint,
			RedirectURL: fmt.Sprintf(
				"https://%s/authorize/github",
				host,
			),
			Scopes: []string{
				"user:email",
				"write:repo_hook",
				"read:repo_hook",
				"repo_deployment",
			},
		},
	}

	r := chi.NewRouter()
	r.Use(web.AccessLog(logger))
	r.Use(web.Metrics())
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if GithubLoggedIn(r) {
			w.Write(index)
		} else {
			w.Write(loginHTML)
		}
	})

	conn, err := grpc.Dial(
		lokiAddr,
		grpc.WithInsecure(),
		grpc.WithUserAgent(fmt.Sprintf(
			"hooks (%s; %s; %s) grpc-go/%s",
			runtime.Version(),
			runtime.GOOS,
			runtime.GOARCH,
			grpc.Version,
		)),
	)
	if err != nil {
		logger.WithError(err).Fatal("could not dial loki's grpc endpoint")
	}
	defer conn.Close()
	pusher := logproto.NewPusherClient(conn)

	r.Get("/login/github", gh.Login)
	r.Post("/authorize/github", gh.Authorize)
	r.Post("/logout/github", gh.SignOut)
	r.Post("/hooks/github", callback())
	r.Post("/hooks/minio/logs", minioLoggingHookHandler[*MinioLogEntry](pusher))
	r.Post("/hooks/minio/audit", minioLoggingHookHandler[*MinioAuditEntry](pusher))
	r.Handle("/metrics", web.MetricsHandler())

	addr := fmt.Sprintf(":%d", port)
	if err = web.ListenAndServe(addr, r); err != nil {
		logger.WithError(err).Fatal("listen and serve failed")
	}
}

const callbackSecret = "e5c172c9302d3bec1ae54314d2f1be70301cca1c289afb94adac83066128c31e"

func callback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			SendError(w, http.StatusBadRequest, nil)
			return
		}
		ct := r.Header.Get("Content-Type")
		spec, err := ParseHookSpec(r.Header)
		if err != nil {
			SendError(w, http.StatusBadRequest, err, "failed to parse event")
			return
		}
		logger.WithFields(logrus.Fields{
			"user-agent":       r.UserAgent(),
			"content-type":     ct,
			"github-delivery":  spec.Delivery,
			"github-event":     spec.Event,
			"hook-id":          spec.ID,
			"hook-target-id":   spec.TargetID,
			"hook-target-type": spec.TargetType,
		}).Info("webhook callback")

		payload, err := github.ValidatePayload(r, []byte(callbackSecret))
		if err != nil {
			SendError(w, http.StatusBadRequest, err)
			return
		}
		event, err := github.ParseWebHook(github.WebHookType(r), payload)
		if err != nil {
			SendError(w, http.StatusBadRequest, err)
			return
		}
		switch ev := event.(type) {
		case *github.PushEvent:
			fmt.Println(ev)
		case *github.DeploymentEvent:
			fmt.Println(ev)
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

type HookSpec struct {
	Delivery   string
	ID         int
	Event      string
	TargetID   int
	TargetType string
	Signature  []byte
}

func ParseHookSpec(h http.Header) (*HookSpec, error) {
	id, err := strconv.ParseInt(h.Get("X-GitHub-Hook-ID"), 10, 64)
	if err != nil {
		return nil, err
	}
	target, err := strconv.ParseInt(h.Get("X-GitHub-Hook-Installation-Target-ID"), 10, 64)
	if err != nil {
		return nil, err
	}
	signature, err := getSignature256(h)
	if err != nil {
		return nil, err
	}
	return &HookSpec{
		ID:         int(id),
		TargetID:   int(target),
		Event:      h.Get("X-GitHub-Event"),
		Delivery:   h.Get("X-GitHub-Delivery"),
		TargetType: h.Get("X-GitHub-Hook-Installation-Target-Type"),
		Signature:  signature,
	}, nil
}

func getSignature256(h http.Header) ([]byte, error) {
	s := h.Get("X-Hub-Signature-256")
	if len(s) == 0 {
		return nil, errors.New("no signature")
	}
	s = strings.Replace(s, "sha256=", "", 1)
	return hex.DecodeString(s)
}

func verifySignature(body, signature []byte) (err error) {
	mac := hmac.New(sha256.New, []byte(callbackSecret))
	if _, err = mac.Write(body); err != nil {
		return err
	}
	if !bytes.Equal(mac.Sum(nil), signature) {
		return errors.New("invalid signature")
	}
	return nil
}

func redirect(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusFound)
	logger.WithField("location", location).Info("redirecting")
}

type response struct {
	http.ResponseWriter
	status int
}

func (r *response) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func getState() string {
	var b [32]byte
	rand.Read((b[:]))
	return hex.EncodeToString(b[:])
}

func SendError(w http.ResponseWriter, status int, err error, msgs ...string) {
	msg := strings.Join(msgs, ": ")
	logger.WithFields(logrus.Fields{
		"status": status,
		"error":  err,
	}).Error(msg)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func validateEnv() error {
	for _, key := range []string{
		"GITHUB_CLIENT_ID",
		"GITHUB_CLIENT_SECRET",
		"GH_HOOK_CALLBACK_HOST",
	} {
		k, ok := os.LookupEnv(key)
		if !ok || len(k) == 0 {
			return fmt.Errorf("could not find environment variable %q", key)
		}
	}
	return nil
}
