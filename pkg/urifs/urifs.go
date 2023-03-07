package urifs

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"harrybrown.com/pkg/log"
)

var logger = log.GetLogger()

func Open(uri *url.URL) (io.ReadCloser, error) {
	return openURI(logger, uri)
}

func openURI(logger log.FieldLogger, uri *url.URL) (io.ReadCloser, error) {
	switch uri.Scheme {
	case "":
		fallthrough // defaults to "file"
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
		region := getenv("AWS_S3_REGION", getenv("AWS_REGION", "us-east-1"))
		session, err := s3SessionFromURI(logger, uri, region)
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

func s3SessionFromURI(logger log.FieldLogger, uri *url.URL, region string) (client.ConfigProvider, error) {
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
		DisableSSL:       aws.Bool(getenv("S3_ALLOW_INSECURE", "false") == "true"),
		S3ForcePathStyle: aws.Bool(true),
		Logger:           aws.LoggerFunc(logger.WithField("client", "s3").Info),
		Credentials: credentials.NewChainCredentials(
			[]credentials.Provider{
				&credentials.EnvProvider{},
				&credentials.StaticProvider{Value: credentials.Value{
					AccessKeyID:     uri.User.Username(),
					SecretAccessKey: pw,
					SessionToken:    "",
				}},
			},
		),
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

func getenv(key, defaultValue string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return val
}
