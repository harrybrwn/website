package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sJson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	AppLabel    = "app"
	DefaultRepo = "10.0.0.11:5000"
)

type GlobalAppSettings struct {
	ImagePullPolicy corev1.PullPolicy `json:"image_pull_policy"`
}

func (gas *GlobalAppSettings) InitDefaults() {
	if len(gas.ImagePullPolicy) == 0 {
		gas.ImagePullPolicy = corev1.PullAlways
	}
}

type AppConfig struct {
	Name      string
	Port      int
	Image     *ImageConfig
	Size      Size
	Args      []string
	Namespace string

	SkipPrometheus bool `json:"skip_prometheus"`
}

type Size uint8

const (
	Unknown Size = iota
	Big
	Med
	Sml
)

func NewSize(v string) Size {
	switch strings.ToLower(strings.Trim(v, " \n\t\r")) {
	case "l", "b", "lg", "bg", "big", "large":
		return Big
	case "m", "med", "medium":
		return Med
	case "s", "sm", "sml", "small":
		return Sml
	default:
		return Unknown
	}
}

func (s Size) String() string {
	switch s {
	case Big:
		return "big"
	case Med:
		return "medium"
	case Sml:
		return "small"
	default:
		return "<unknown>"
	}
}

func (s *Size) UnmarshalJSON(b []byte) error {
	var (
		start = 0
		end   = len(b)
	)
	if b[0] == '"' {
		start += 1
	}
	if b[len(b)-1] == '"' {
		end -= 1
	}
	*s = NewSize(string(b[start:end]))
	return nil
}

func (c *AppConfig) InitDefaults(gas *GlobalAppSettings) {
	if c.Image == nil {
		c.Image = &ImageConfig{}
	}
	if len(c.Image.Name) == 0 {
		c.Image.Name = c.Name
	}
	if len(c.Image.Repo) == 0 {
		c.Image.Repo = DefaultRepo
	}
	if len(c.Image.Tag) == 0 {
		c.Image.Tag = "latest"
	}
}

func (c *AppConfig) ImageName() string {
	if c.Image == nil {
		return fmt.Sprintf("%s/harrybrwn/%s:latest", DefaultRepo, c.Name)
	}
	return fmt.Sprintf("%s/%s:%s", c.Image.Repo, c.Image.Name, c.Image.Tag)
}

type ImageConfig struct {
	Name string
	Tag  string
	Repo string
}

func (c *AppConfig) ns() string {
	ns := "default"
	if len(c.Namespace) > 0 {
		ns = c.Namespace
	}
	return ns
}

func (c *AppConfig) resources() corev1.ResourceRequirements {
	var (
		cpuReq, cpuLim int64
		memReq, memLim string
	)
	switch c.Size {
	case Big:
		memLim, cpuLim = "512Mi", 250
		memReq, cpuReq = "256Mi", 100
	case Med:
		memLim, cpuLim = "256Mi", 100
		memReq, cpuReq = "128Mi", 50
	case Sml:
		memLim, cpuLim = "128Mi", 50
		memReq, cpuReq = "64Mi", 10
	case Unknown:
		return corev1.ResourceRequirements{}
	}
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse(memLim),
			corev1.ResourceCPU:    *resource.NewMilliQuantity(cpuLim, resource.BinarySI),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse(memReq),
			corev1.ResourceCPU:    *resource.NewMilliQuantity(cpuReq, resource.BinarySI),
		},
	}
}

func (c *AppConfig) deployment(settings *GlobalAppSettings) appsv1.Deployment {
	annotations := map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   strconv.Itoa(c.Port),
	}
	if c.SkipPrometheus {
		annotations = nil
	}
	d := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.ns(),
			Labels: map[string]string{
				AppLabel: c.Name,
			},
			// CreationTimestamp: metav1.NewTime(time.Now()),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: asPtr(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{AppLabel: c.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						AppLabel: c.Name,
					},
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            c.Name,
						Image:           c.ImageName(),
						ImagePullPolicy: corev1.PullAlways,
						Args:            c.Args,
						Ports: []corev1.ContainerPort{{
							Name:          "http",
							ContainerPort: int32(c.Port),
						}},
						Resources: c.resources(),
					}},
				},
			},
		},
	}
	return d
}

func (c *AppConfig) service() corev1.Service {
	return corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.ns(),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				AppLabel: c.Name,
			},
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       int32(c.Port),
				TargetPort: intstr.FromString("http"),
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}
}

type Config struct {
	GlobalAppSettings `json:",inline"`
	Apps              []*AppConfig
}

func main() {
	var (
		out        string
		configFile = "k8s-generate.yml"
	)
	flag.StringVar(&out, "out", out, "output file or directory")
	flag.StringVar(&configFile, "c", configFile, "config file to read")
	flag.Parse()

	if out == "" {
		log.Fatal("must specify output dir with -out. use '-' to output to stdout")
	}

	serializer := k8sJson.NewSerializerWithOptions(
		k8sJson.DefaultMetaFactory, nil, nil,
		k8sJson.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)

	var config Config
	configBytes, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		log.Fatal(err)
	}
	config.InitDefaults()

	for _, conf := range config.Apps {
		if conf.Name == "" {
			log.Fatal("'name' is a required configuration attribute")
		}
		conf.InitDefaults(&config.GlobalAppSettings)
		deploy := conf.deployment(&config.GlobalAppSettings)
		service := conf.service()
		var output io.Writer
		if out != "-" {
			err = os.MkdirAll(filepath.Join(out, conf.Name), os.FileMode(0755))
			if err != nil {
				log.Fatal("failed to create directory")
			}
		}

		if out == "-" {
			output = os.Stdout
		} else {
			f, err := os.OpenFile(
				filepath.Join(out, conf.Name, "deployment.yml"),
				os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
				os.FileMode(0644),
			)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			output = f
		}
		output.Write([]byte("---"))
		err = serializer.Encode(&deploy, output)
		if err != nil {
			log.Fatal(err)
		}

		if out == "-" {
			output = os.Stdout
		} else {
			f, err := os.OpenFile(
				filepath.Join(out, conf.Name, "service.yml"),
				os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
				os.FileMode(0644),
			)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			output = f
		}
		output.Write([]byte("---"))
		err = serializer.Encode(&service, output)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func asPtr[T any](v T) *T {
	return &v
}
