package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"reflect"

	"github.com/imdario/mergo"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
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

type Config struct {
	S3 S3Config `json:"s3" yaml:"s3"`
	DB DBConfig `json:"db" yaml:"db"`
}

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

type DBConfig struct {
	Host      string `json:"host" yaml:"host"`
	Port      string `json:"port" yaml:"port"`
	RootUser  string `json:"root_user" yaml:"root_user"`
	Password  string `json:"password" yaml:"password"`
	Users     []*DBUser
	Databases []*struct {
		Name  string
		Owner string
	}
}

type DBUser struct {
	Name       string
	Password   string
	SuperUser  bool
	CreateDB   bool
	CreateRole bool
}

func (db *DBConfig) init() {
	if db.Host == "" {
		db.Host = os.Getenv("POSTGRES_HOST")
	}
	if db.Port == "" {
		db.Port = os.Getenv("POSTGRES_PORT")
	}
	if db.Port == "" {
		db.Port = "5432"
	}
	if db.RootUser == "" {
		db.RootUser = os.Getenv("POSTGRES_USER")
	}
	if db.Password == "" {
		db.Password = os.Getenv("POSTGRES_PASSWORD")
	}
}

const (
	pqDuplicateObject   = "42710"
	pqDuplicateDatabase = "42P04"
)

func (db *DBConfig) Provision(ctx context.Context) error {
	os.Unsetenv("PGSERVICEFILE")
	os.Unsetenv("PGSERVICE")
	d, err := sql.Open("postgres", db.uri())
	if err != nil {
		return err
	}
	defer d.Close()

	for _, user := range db.Users {
		query := fmt.Sprintf(
			`CREATE ROLE "%s" WITH PASSWORD '%s' `,
			user.Name,
			user.Password)
		if user.SuperUser {
			query += "SUPERUSER "
		}
		if user.CreateDB {
			query += "CREATEDB "
		}
		if user.CreateRole {
			query += "CREATEROLE "
		}
		_, err = d.ExecContext(ctx, query)
		switch e := err.(type) {
		case nil:
			continue
		case *pq.Error:
			if e.Code == pqDuplicateObject {
				continue
			}
			return e
		default:
			return err
		}
	}

	for _, database := range db.Databases {
		if database.Owner == "" {
			return errors.New("each database must have an owner")
		}
		query := fmt.Sprintf(
			`CREATE DATABASE "%s" OWNER '%s'`,
			database.Name,
			database.Owner)
		_, err := d.ExecContext(ctx, query)
		switch e := err.(type) {
		case nil:
			continue
		case *pq.Error:
			if e.Code == pqDuplicateDatabase {
				continue
			}
			return err
		default:
			return err
		}
	}
	return nil
}

func (db *DBConfig) uri() string {
	u := url.URL{
		Scheme:   "postgres",
		Host:     db.Host,
		User:     url.UserPassword(db.RootUser, db.Password),
		Path:     "/",
		RawQuery: "sslmode=disable",
	}
	return u.String()
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

func copyVal(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	var cp reflect.Value

	switch v.Kind() {
	case reflect.Array:
		t := reflect.ArrayOf(v.Len(), v.Type().Elem())
		cp = reflect.New(t).Elem()
		reflect.Copy(cp, v)
	case reflect.Slice:
		cp = reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		reflect.Copy(cp, v)
	case reflect.Struct:
		cp = reflect.New(v.Type()).Elem()

		for i := 0; i < v.NumField(); i++ {
			vf := v.Field(i)
			cf := cp.Field(i)
			switch vf.Kind() {
			case reflect.Ptr:
				if vf.IsNil() {
					continue
				}
				fieldcopy := copyVal(vf.Elem())
				cf = reflect.New(vf.Elem().Type())
				cf.Elem().Set(fieldcopy)
				cp.Field(i).Set(cf)
			default:
				cp.Field(i).Set(copyVal(vf))
			}
		}
	case reflect.Map:
		cp = reflect.MakeMap(v.Type())
		for _, key := range v.MapKeys() {
			cp.SetMapIndex(key, copyVal(v.MapIndex(key)))
		}
	default:
		cp = reflect.New(v.Type()).Elem()
		cp.Set(v)
	}
	return cp
}

var errMismatchedTypes = errors.New("mismatched types")

// merge the fields of src into dst if they have not
// already been set.
func merge(dst, src reflect.Value) error {
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}
	if dst.Kind() == reflect.Ptr {
		dst = dst.Elem()
	}
	if dst.Kind() != src.Kind() {
		return errMismatchedTypes
	}

	var err error
	switch dst.Kind() {
	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			sf := src.Field(i) // source field
			df := dst.Field(i) // dest field

			// If there is no value to set, then skip it
			if sf.IsZero() {
				continue
			}
			if sf.Kind() == reflect.Ptr {
				// Copy of nil is useless
				if sf.IsNil() {
					continue
				}
				if df.IsNil() {
					df = reflect.New(sf.Elem().Type())
				}
			}
			err = merge(df, sf)
			if err != nil {
				return err
			}
			if df.CanSet() {
				dst.Field(i).Set(df)
			}
		}

	case reflect.Map:
		var dstval, srcval reflect.Value
		if dst.IsNil() {
			dst.Set(reflect.MakeMap(src.Type()))
		}
		for _, key := range src.MapKeys() {
			dstval = dst.MapIndex(key)
			srcval = src.MapIndex(key)
			// if the key is not in dst, then
			// copy the value from the source map
			// and insert it into the dest
			if !dstval.IsValid() {
				dstval = copyVal(srcval)
				if srcval.Kind() == reflect.Ptr {
					dstval = dstval.Addr()
				}
			} else {
				err = merge(dstval, srcval)
				if err != nil {
					return err
				}
			}
			dst.SetMapIndex(key, dstval)
		}

	case reflect.Slice:
		if dst.IsZero() {
			dst.Set(src)
		} else if dst.CanSet() {
			dst.Set(reflect.AppendSlice(dst, src))
		} else {
			return errors.New("can't append slice")
		}
	case reflect.Array:
		return errors.New("can merge arrays")
	default:
		if dst.IsZero() {
			dst.Set(src)
		}
	}
	return nil
}
