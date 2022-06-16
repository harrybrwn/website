package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/imdario/mergo"
	_ "github.com/lib/pq"
	"github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/cobra"
)

func main() {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
	}
}

func NewRootCmd() *cobra.Command {
	var (
		configFiles = []string{"config/provision.json"}
		envFile     string
		cli         Cli
	)
	c := &cobra.Command{
		Use: "provision",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cli.config.S3.init()
			cli.config.DB.init()
			return cli.readConfig(configFiles)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			err := cli.config.S3.Provision(ctx, cli.admin, cli.client)
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
	flg.StringVar(&cli.config.S3.AccessKey, "s3-access-key", "", "access key for s3 object storage")
	flg.StringVar(&cli.config.S3.SecretKey, "s3-secret-key", "", "secret key for s3 object storage")
	flg.StringVar(&cli.config.S3.Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "endpoint for s3 object storage")
	flg.StringVar(&envFile, "env-file", "", "load environment variables from a file")
	return c
}

type Cli struct {
	client *minio.Client
	admin  *madmin.AdminClient
	config Config
}

type Config struct {
	S3 S3Config `json:"s3" yaml:"s3"`
	DB DBConfig `json:"db" yaml:"db"`
}

func (cli *Cli) readConfig(files []string) error {
	// dst := reflect.ValueOf(&cli.config)
	for _, file := range files {
		if file == "-" {
		}
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		var config Config
		err = json.NewDecoder(f).Decode(&config)
		if err != nil {
			return err
		}
		// err = merge(dst, reflect.ValueOf(config))
		// err = mergo.Merge(&cli.config, &config, mergo.WithAppendSlice)
		err = mergo.Merge(
			&cli.config,
			&config,
			// mergo.WithTransformers(transformerFunc(dbUserTransformer)),
			mergo.WithAppendSlice,
		)
		if err != nil {
			return err
		}
		cli.client, err = s3Client(&cli.config.S3)
		if err != nil {
			return err
		}
		cli.admin, err = minioAdmin(&cli.config.S3)
		if err != nil {
			return err
		}
	}
	return nil
}

func dbUserTransformer(typ reflect.Type) func(dst, src reflect.Value) error {
	fn := func(dst, src reflect.Value) error {
		names := make(map[string][]*DBUser)
		for i := 0; i < src.Len(); i++ {
			v := src.Index(i)
			name := v.FieldByName("Name").String()
			if u, ok := names[name]; ok {
				names[name] = append(u, v.Interface().(*DBUser))
			} else {
				names[name] = []*DBUser{v.Interface().(*DBUser)}
			}
		}
		return nil
	}
	if typ == reflect.TypeOf([]*DBUser{}) {
		return fn
	}
	return nil
}

type transformerFunc func(typ reflect.Type) func(dst, src reflect.Value) error

func (tf transformerFunc) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	return tf(typ)
}
