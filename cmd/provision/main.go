package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	_ "github.com/lib/pq"
	"github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/provision"
)

var logger = log.New(
	log.WithEnv(),
	log.WithFormat(log.TextFormat),
)

func main() {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
	}
}

func NewRootCmd() *cobra.Command {
	var (
		configFiles   = []string{"config/provision.json"}
		extraPolicies []string
		envFile       string
		cli           Cli
	)
	c := &cobra.Command{
		Use: "provision",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cli.config.S3.Init()
			cli.config.DB.Init()
			return cli.readConfig(configFiles)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			err := cli.config.S3.Provision(ctx, logger, cli.admin, cli.client)
			if err != nil {
				return err
			}
			err = cli.config.DB.Provision(ctx)
			if err != nil {
				return err
			}
			return nil
		},
	}
	c.AddCommand(
		&cobra.Command{Use: "config", RunE: func(cmd *cobra.Command, args []string) error {
			b, err := json.MarshalIndent(cli.config, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", b)
			return nil
		}},
	)
	flg := c.PersistentFlags()
	flg.StringArrayVarP(&configFiles, "config", "c", configFiles, "specify the config file")
	flg.StringArrayVar(&extraPolicies, "extra-policy", extraPolicies, "add extra AWS Access policies from a json file")
	flg.StringVar(&cli.config.S3.AccessKey, "s3-access-key", "", "access key for s3 object storage")
	flg.StringVar(&cli.config.S3.SecretKey, "s3-secret-key", "", "secret key for s3 object storage")
	flg.StringVar(&cli.config.S3.Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "endpoint for s3 object storage")
	flg.StringVar(&envFile, "env-file", "", "load environment variables from a file")
	return c
}

type Cli struct {
	client *minio.Client
	admin  *madmin.AdminClient
	config provision.Config
}

func (cli *Cli) readConfig(files []string) error {
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		err = cli.config.ApplyFile(f)
		if err != nil {
			return err
		}
	}
	var err error
	cli.client, err = s3Client(&cli.config.S3)
	if err != nil {
		return err
	}
	cli.admin, err = minioAdmin(&cli.config.S3)
	if err != nil {
		return err
	}
	return nil
}

func s3Client(cfg *provision.S3Config) (*minio.Client, error) {
	return minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       false,
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupAuto,
	})
}

func minioAdmin(cfg *provision.S3Config) (*madmin.AdminClient, error) {
	return madmin.New(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, false)
}

func dbUserTransformer(typ reflect.Type) func(dst, src reflect.Value) error {
	fn := func(dst, src reflect.Value) error {
		names := make(map[string][]*provision.DBUser)
		for i := 0; i < src.Len(); i++ {
			v := src.Index(i)
			name := v.FieldByName("Name").String()
			if u, ok := names[name]; ok {
				names[name] = append(u, v.Interface().(*provision.DBUser))
			} else {
				names[name] = []*provision.DBUser{v.Interface().(*provision.DBUser)}
			}
		}
		return nil
	}
	if typ == reflect.TypeOf([]*provision.DBUser{}) {
		return fn
	}
	return nil
}

type transformerFunc func(typ reflect.Type) func(dst, src reflect.Value) error

func (tf transformerFunc) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	return tf(typ)
}
