package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	"harrybrown.com/pkg/log"
)

type S3Config struct {
	AccessKey string
	SecretKey string
	Endpoint  string
	Buckets   []*struct {
		Name   string
		Policy string `json:"policy,omitempty"`
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
		Effect    string
		Principal struct {
			AWS     []string `json:",omitempty"`
			Service string   `json:",omitempty"`
		} `json:",omitempty"`
		Action    []string
		Resource  []string
		Condition struct {
			StringEquals struct {
				S3XAmzAcl []string `json:"s3:x-amx-acl,omitempty"`
				S3Prefix  []string `json:"s3:prefix,omitempty"`
			} `json:",omitempty"`
			IpAddress    IPAddressCondition `json:",omitempty"`
			NotIpAddress IPAddressCondition `json:",omitempty"`
			StringLike   struct {
				AWSReferer []string `json:"aws:Referer,omitempty"`
			} `json:",omitempty"`
			Null struct {
				AWSMultiFactorAuthAge bool `json:"aws:MultiFactorAuthAge,omitempty"`
			} `json:",omitempty"`
			NumericGreaterThan struct {
				AWSMultiFactorAuthAge int `json:"aws:MultiFactorAuthAge,omitempty"`
			} `json:",omitempty"`
		} `json:",omitempty"`
	}
}

type IPAddressCondition struct {
	AWSSourceIP string `json:"aws:SourceIp,omitempty"`
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

const supportedPolicyVersion = "2012-10-17"

func (s3 *S3Config) Validate() error {
	for name, p := range s3.Policies {
		if p.Version != supportedPolicyVersion {
			return fmt.Errorf(
				"%q is not a supported policy version. Use %q",
				p.Version, supportedPolicyVersion,
			)
		}
		if p.Statement == nil {
			return fmt.Errorf("policy %q has no statement", name)
		}
		for _, statement := range p.Statement {
			if statement.Effect == "" {
				return errors.New("statement has no \"Effect\" field")
			}
		}
	}
	return nil
}

func (s3 *S3Config) Provision(ctx context.Context, admin *madmin.AdminClient, client *minio.Client) error {
	var err error
	if err = s3.Validate(); err != nil {
		return err
	}
	for _, b := range s3.Buckets {
		err = client.MakeBucket(ctx, b.Name, minio.MakeBucketOptions{})
		switch e := err.(type) {
		case nil:
			break
		case minio.ErrorResponse:
			if e.Code == "BucketAlreadyOwnedByYou" {
				logger.WithFields(logMinioErrorFields(e)).Warn("bucket already exists")
				break
			}
		default:
			return errors.Wrap(err, "failed to create s3 bucket")
		}
		var raw []byte
		if b.Policy == "" {
			// Delete any policies on the bucket
			raw = []byte{}
			err = client.SetBucketPolicy(ctx, b.Name, "")
			if err != nil {
				return errors.Wrap(err, "failed to set bucket policy")
			}
			continue
		} else {
			// Get the policy from our config
			p, ok := s3.Policies[b.Policy]
			if !ok {
				return fmt.Errorf("policy %q not defined in configuration", b.Policy)
			}
			raw, err = json.Marshal(p)
			if err != nil {
				return err
			}
		}
		err = client.SetBucketPolicy(ctx, b.Name, string(raw))
		if err != nil {
			return errors.Wrap(err, "failed to set bucket policy")
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

	// Save a list of user's Access Keys so we can use then for creating groups.
	groupUsers := make(map[string][]string)

	for _, user := range s3.Users {
		err = admin.AddUser(ctx, user.AccessKey, user.SecretKey)
		if err != nil {
			return errors.Wrap(err, "failed to add new user")
		}
		// Set multiple policies with comma separated list.
		// https://github.com/minio/console/blob/ba4103e03f62cb33a12fe6348727c1ecf04ba037/restapi/admin_policies.go#L659
		err = admin.SetPolicy(ctx, strings.Join(user.Policies, ","), user.AccessKey, false)
		if err != nil {
			return errors.Wrap(err, "failed to set user policy")
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

func logMinioErrorFields(e minio.ErrorResponse) log.Fields {
	return log.Fields{
		"code":    e.Code,
		"bucket":  e.BucketName,
		"message": e.Message,
		"region":  e.Region,
		"server":  e.Server,
		"status":  e.StatusCode,
	}
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
