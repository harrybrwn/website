package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/logging"
	"github.com/obalunenko/getenv"
	"github.com/obalunenko/getenv/option"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var Debug bool

func main() {
	var (
		config = Config{
			AccountID:  getenv.EnvOrDefault("GEOIPUPDATE_ACCOUNT_ID", 0),
			LicenseKey: getenv.EnvOrDefault("GEOIPUPDATE_LICENSE_KEY", ""),
			EditionIDs: getenv.EnvOrDefault(
				"GEOIPUPDATE_EDITION_IDS", ([]string)(nil),
				option.WithSeparator(","),
			),
		}
		ctx       = context.Background()
		bucket    = "geoip"
		baseDir   = "/opt/geoip"
		modeFlag  = "file"
		env       = "dev"
		namespace = readNamespaceOr("default")
		logger    = logrus.New()

		labelSelector, podName string
	)
	flag.IntVar(&config.AccountID, "account-id", config.AccountID, "")
	flag.StringArrayVar(&config.EditionIDs, "edition", config.EditionIDs, "")
	flag.StringVar(&config.LicenseKey, "license-key", config.LicenseKey, "")
	flag.StringVarP(&modeFlag, "mode", "m", modeFlag, "switch the data persistence mode")
	flag.StringVar(&bucket, "bucket", bucket, "")
	flag.StringVar(&baseDir, "base-dir", baseDir, "")
	flag.StringVarP(&namespace, "namespace", "n", namespace, "select target namespace for finding pods to clobber")
	flag.StringVarP(&labelSelector, "selector", "l", labelSelector, "selector to use for finding pods to clobber")
	flag.StringVar(&podName, "pod-name", podName, "delete one pod")
	flag.StringVar(&env, "env", env, "use a different environment")
	flag.BoolVar(&Debug, "debug", Debug, "run with debug logs")
	test := false
	flag.BoolVarP(&test, "test", "t", test, "")
	flag.Parse()

	k8sconfig, err := k8sConfig()
	if err != nil {
		logger.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(k8sconfig)
	if err != nil {
		logger.Fatal(err)
	}

	if test {
		podClient := clientset.CoreV1().Pods("")
		pods, err := podClient.List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Fatal(err)
		}
		for _, pod := range pods.Items {
			// fmt.Printf("%+v\n", pod)
			fmt.Println(pod.Name, pod.Annotations)
		}
		return
	}

	mode := ModeFromString(modeFlag)
	if mode == ModeInvalid {
		logger.Fatal("invalid storage mode")
	}

	cfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(minioEndpointResolver)),
		awsconfig.WithDefaultRegion("us-east-2"),
		awsconfig.WithLogger(logging.NewStandardLogger(os.Stdout)),
	)
	if err != nil {
		log.Fatal(err)
	}

	// TODO
	// 1. Create fs.FS impl for s3 and local files
	// 2. Create a mock for fs.FS
	// 3. Generate a gomock for s3 then test
	// 4. Add metadata to k8s api
	// 5. Option for injecting a configmap with the uri of the new object.
	// 6. Figure out how the S3 key should be in different environments, we
	// 	  don't want to store a copy for both 'stg' and 'prd'
	// 7. GeoLite2 databases can have corrupted data, we should have some smoke
	// 	  tests to make sure the data we're uploading is valid.

	var (
		wg     sync.WaitGroup
		client = NewClient(&config, &cfg, WithEnv(env), WithMode(mode))
		errs   = make(chan error)
	)
	wg.Add(len(config.EditionIDs))
	go func() {
		wg.Wait()
		close(errs)
	}()
	for _, ed := range config.EditionIDs {
		childCtx, done := context.WithTimeout(ctx, time.Minute)
		go func(ctx context.Context, ed string) {
			defer wg.Done()
			defer done()
			logger.WithField("edition_id", ed).Info("fetching file")
			r, err := client.Download(ctx, ed)
			if err != nil {
				errs <- err
				return
			}
			defer r.Close()

			var iwg sync.WaitGroup // inner wait group
			for _, f := range r.Files {
				iwg.Add(1)
				go func(file file) {
					defer iwg.Done()
					var err error
					if mode == ModeFile {
						err = client.write(ctx, baseDir, &file)
					} else if mode == ModeS3 {
						err = client.upload(ctx, bucket, &file)
					} else {
						errs <- fmt.Errorf("invalid upload mode %s", mode)
						return
					}
					if errors.Is(err, fs.ErrExist) {
						logger.Printf("%q already exists", file.name)
						return
					}
					if err != nil {
						errs <- err
						return
					}
				}(f)
			}
			iwg.Wait()
			logger.WithField("edition_id", ed).Info("done")
		}(childCtx, ed)
	}
	for e := range errs {
		// Hulk smash the universe when we encounter the first error.
		if e != nil {
			logger.Fatalf("Error while downloading data: %v", e)
		}
	}

	// Restart the pod
	podClient := clientset.CoreV1().Pods(namespace)
	if len(podName) > 0 {
		logger.Printf("deleting pod %q", podName)
		err = podClient.Delete(ctx, podName, metav1.DeleteOptions{})
		if err != nil {
			logger.Fatal(err)
		}
	} else if len(labelSelector) > 0 {
		logger.Printf("deleting pods based on %q", labelSelector)
		pods, err := podClient.List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			logger.Fatal(err)
		}
		for _, pod := range pods.Items {
			logger.WithField("pod", pod.Name).Info("deleting pod")
			err = podClient.Delete(ctx, pod.Name, metav1.DeleteOptions{})
			if err != nil {
				logger.Fatal(err)
			}
		}
	} else {
		logger.Info("Skipping pod deletion, use --pod-name or --selector to delete a pod when the download is finished")
	}
}

type Mode uint8

const (
	ModeInvalid Mode = iota
	ModeFile
	ModeS3
)

func ModeFromString(s string) Mode {
	switch strings.ToLower(s) {
	case "file":
		return ModeFile
	case "s3":
		return ModeS3
	default:
		return ModeInvalid
	}
}

func (m Mode) String() string {
	switch m {
	case ModeFile:
		return "file"
	case ModeS3:
		return "s3"
	}
	return "invalid"
}

func minioEndpointResolver(service, region string, opts ...any) (aws.Endpoint, error) {
	if service == s3.ServiceID {
		url, ok := os.LookupEnv("AWS_ENDPOINT_URL")
		if ok {
			return aws.Endpoint{
				PartitionID:       "aws",
				URL:               url,
				SigningRegion:     region,
				HostnameImmutable: true,
				Source:            aws.EndpointSourceCustom,
			}, nil

		}
	}
	return aws.Endpoint{}, &aws.EndpointNotFoundError{}
}

func k8sConfig() (*restclient.Config, error) {
	f := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	if !exists(f) {
		f = "" // will resolve to rest.InClusterConfig()
	}
	return clientcmd.BuildConfigFromFlags("", f)
}

func readNamespaceOr(defValue string) string {
	f, err := os.Open("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return defValue
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return defValue
	}
	return strings.Trim(string(b), " \n\t")
}

func exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}
