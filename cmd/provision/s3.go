package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
)

type S3Config struct {
	AccessKey string
	SecretKey string
	Endpoint  string
	Buckets   []*struct {
		Name string
	}
	// Mapping of names to policies to create
	Policies map[string]*S3Policy `json:"policies" yaml:"policies"`
	Groups   []*struct {
		Name     string
		Policies []string
	} `json:"groups" yaml:"groups"`
	Users []*struct {
		AccessKey string
		SecretKey string
		Policies  []string
		Groups    []string
	} `json:"users" yaml:"users"`
}

type S3Policy struct {
	Version   string
	Statement []*struct {
		Effect   string
		Action   []string
		Resource []string
	}
}

func (s3 *S3Config) init() {
	if s3.AccessKey == "" {
		s3.AccessKey = os.Getenv("S3_ACCESS_KEY")
	}
	if s3.SecretKey == "" {
		s3.SecretKey = os.Getenv("S3_SECRET_KEY")
	}
	if s3.Endpoint == "" {
		s3.Endpoint = os.Getenv("S3_ENDPOINT")
	}
	if s3.Endpoint == "" {
		s3.Endpoint = "localhost:9000"
	}
}

func (s3 *S3Config) Provision(ctx context.Context, admin *madmin.AdminClient, client *minio.Client) error {
	var err error
	for _, b := range s3.Buckets {
		err = client.MakeBucket(ctx, b.Name, minio.MakeBucketOptions{})
		switch e := err.(type) {
		case nil:
			continue
		case minio.ErrorResponse:
			if e.Code == "BucketAlreadyOwnedByYou" {
				continue
			}
		default:
			return errors.Wrap(err, "failed to create s3 bucket")
		}
	}
	for name, p := range s3.Policies {
		raw, err := json.Marshal(p)
		if err != nil {
			return err
		}
		if err = admin.AddCannedPolicy(ctx, name, raw); err != nil {
			return errors.Wrap(err, "failed to create minio policy")
		}
	}

	groupUsers := make(map[string][]string)
	for _, user := range s3.Users {
		err = admin.AddUser(ctx, user.AccessKey, user.SecretKey)
		if err != nil {
			return errors.Wrap(err, "failed to add new user")
		}
		for _, policy := range user.Policies {
			err = admin.SetPolicy(ctx, policy, user.AccessKey, false)
			if err != nil {
				return errors.Wrap(err, "failed to set user policy")
			}
		}
		// Save the user access ids for later when we create the groups.
		for _, g := range user.Groups {
			if list, ok := groupUsers[g]; ok {
				groupUsers[g] = append(list, user.AccessKey)
			} else {
				groupUsers[g] = []string{user.AccessKey}
			}
		}
	}

	for _, group := range s3.Groups {
		accessIDs, ok := groupUsers[group.Name]
		if !ok {
			continue
		}
		err := admin.UpdateGroupMembers(ctx, madmin.GroupAddRemove{
			Group:    group.Name,
			Members:  accessIDs,
			Status:   madmin.GroupEnabled,
			IsRemove: false, // we are creating a new group
		})
		if err != nil {
			return errors.Wrap(err, "failed to update user group")
		}
		for _, p := range group.Policies {
			err = admin.SetPolicy(ctx, p, group.Name, true)
			if err != nil {
				return errors.Wrap(err, "failed to set group policy")
			}
		}
	}
	return nil
}

const (
	errBucketExists = iota
)

type S3Error struct {
	err  error
	code int
}

func (s3err *S3Error) Error() string {
	return s3err.err.Error()
}

type S3Admin interface {
	CreateBucket(ctx context.Context, name string) error
}

type minioAdminClient struct {
}

func s3Client(cfg *S3Config) (*minio.Client, error) {
	return minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       false,
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupAuto,
	})
}

func minioAdmin(cfg *S3Config) (*madmin.AdminClient, error) {
	return madmin.New(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, false)
}
