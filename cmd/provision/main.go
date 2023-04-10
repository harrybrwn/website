package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/joho/godotenv"
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func NewRootCmd() *cobra.Command {
	var (
		configFiles   = []string{"config/provision.json"}
		extraPolicies []string
		envFile       string
		cli           Cli
	)
	if exists(".env") {
		godotenv.Load(".env")
	}
	files, ok := configFilesFromEnv()
	if ok {
		configFiles = files
	}
	c := &cobra.Command{
		Use: "provision",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cli.config.S3.Init()
			cli.config.DB.Init()
			err := cli.readConfig(configFiles)
			if err != nil {
				return err
			}
			cli.config.DB.Defaults()
			return nil
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
		NewMigrateCmd(&cli),
		NewConfigCmd(&cli),
		NewValidateCmd(&cli),
		&cobra.Command{
			Use: "test",
			RunE: func(cmd *cobra.Command, args []string) error {
				rawconfig, err := os.ReadFile("config/provision.hcl")
				if err != nil {
					return err
				}
				var c provision.Config
				diags := c.HCLDecode(rawconfig, "config/provision.hcl")
				if diags.HasErrors() {
					return diags
				}
				// fmt.Printf("%+v\n", c)
				// for _, b := range c.S3.Buckets {
				// 	fmt.Printf("%+v\n", b)
				// }
				return nil
			},
		},
	)
	flg := c.PersistentFlags()
	flg.StringArrayVarP(&configFiles, "config", "c", configFiles, "specify the config file")
	flg.StringArrayVar(&extraPolicies, "extra-policy", extraPolicies, "add extra AWS Access policies from a json file")
	flg.StringVar(&cli.config.S3.AccessKey, "s3-access-key", "", "access key for s3 object storage")
	flg.StringVar(&cli.config.S3.SecretKey, "s3-secret-key", "", "secret key for s3 object storage")
	flg.StringVar(&cli.config.S3.Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "endpoint for s3 object storage")
	flg.StringVar(&envFile, "env-file", "", "load environment variables from a file")
	flg.StringVar(&cli.config.DB.Host, "db-host", "", "database host")
	flg.StringVar(&cli.config.DB.Port, "db-port", "", "database port")
	return c
}

type Cli struct {
	client *minio.Client
	admin  *madmin.AdminClient
	config provision.Config
}

func (cli *Cli) readConfig(files []string) (err error) {
	for _, file := range files {
		var rc io.ReadCloser
		if file == "-" {
			var buf bytes.Buffer
			_, err = io.Copy(&buf, os.Stdin)
			if err != nil {
				return err
			}
			rc = io.NopCloser(&buf)
		} else {
			rc, err = os.Open(file)
			if err != nil {
				return err
			}
		}
		defer rc.Close()
		err = cli.config.ApplyFile(rc)
		if err != nil {
			return err
		}
	}
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

func NewConfigCmd(cli *Cli) *cobra.Command {
	c := cobra.Command{
		Use:   "config",
		Short: "Configuration management.",
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := json.MarshalIndent(cli.config, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", b)
			return nil
		},
	}
	return &c
}

func NewValidateCmd(cli *Cli) *cobra.Command {
	c := cobra.Command{
		Use:   "validate",
		Short: "Validate the config file for mistakes.",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				if err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Validation status: failed\n")
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "Validation status: ok\n")
				}
			}()
			if err = cli.config.DB.Validate(); err != nil {
				return err
			}
			if err = cli.config.S3.Validate(); err != nil {
				return err
			}
			return nil
		},
	}
	return &c
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

func configFilesFromEnv() ([]string, bool) {
	filesListEnv := os.Getenv("PROVISION_FILE")
	if filesListEnv == "" {
		return nil, false
	}
	configFiles := strings.Split(filesListEnv, ":")
	return configFiles, true
}

// func dbUserTransformer(typ reflect.Type) func(dst, src reflect.Value) error {
// 	fn := func(dst, src reflect.Value) error {
// 		names := make(map[string][]*provision.DBUser)
// 		for i := 0; i < src.Len(); i++ {
// 			v := src.Index(i)
// 			name := v.FieldByName("Name").String()
// 			if u, ok := names[name]; ok {
// 				names[name] = append(u, v.Interface().(*provision.DBUser))
// 			} else {
// 				names[name] = []*provision.DBUser{v.Interface().(*provision.DBUser)}
// 			}
// 		}
// 		return nil
// 	}
// 	if typ == reflect.TypeOf([]*provision.DBUser{}) {
// 		return fn
// 	}
// 	return nil
// }

// type transformerFunc func(typ reflect.Type) func(dst, src reflect.Value) error

// func (tf transformerFunc) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
// 	return tf(typ)
// }

func exists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
