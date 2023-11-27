package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscale "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type App struct {
	Name      string
	Namespace string
	Image     Image
	Args      []string
	Ports     []Port
	Size      Size
	Replicas  int32
	Resources *Resources
	Scale     *AppScale

	Config          map[string]string
	Secrets         map[string]string
	SkipPrometheus  bool              `json:"skip_prometheus" yaml:"skip_prometheus"`
	ConfigMapName   string            `json:"configmap_name" yaml:"configmap_name"`
	SecretName      string            `json:"secret_name" yaml:"secret_name"`
	ExtraResources  []string          `json:"extra_resources" yaml:"extra_resources"`
	ExtraLabels     map[string]string `json:"extra_labels" yaml:"extra_labels"`
	ResourceProfile ResourceProfile   `json:"-" yaml:"-"`
	Skip            bool
}

type Port struct {
	Type         string
	Port         uint16
	ExternalPort uint16 `yaml:"external_port"`
}

type Resource struct {
	CPU    string
	Memory string
}

type AppScale struct {
	From int32
	To   int32
	When ScaleWhen
}

type ScaleWhen struct {
	CPU    float32
	Memory float32
}

func (r *Resource) Validate() error { return nil }

type Resources struct {
	Limits   *Resource
	Requests *Resource
}

func (rs *Resources) Validate() (err error) {
	if rs.Limits != nil {
		err = rs.Limits.Validate()
		if err != nil {
			return err
		}
	}
	if rs.Requests != nil {
		err = rs.Requests.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *App) Calc(c *K8sGenConfig) {
	a.ConfigMapName = fmt.Sprintf("%s-env", a.Name)
	a.SecretName = fmt.Sprintf("%s-env", a.Name)
	if a.Size == SizeUnknown {
		a.Size = SizeMed
	}
	if len(a.Image.Name) == 0 {
		if len(c.Images.User) > 0 {
			a.Image.Name = fmt.Sprintf("%s/%s", c.Images.User, a.Name)
		} else {
			a.Image.Name = a.Name
		}
	}
	if len(a.Image.Registry) == 0 {
		a.Image.Registry = c.Images.Registry
	}
	if len(a.Image.Tag) == 0 {
		a.Image.Tag = "latest"
	}
	if a.ResourceProfile == nil && c != nil {
		a.ResourceProfile = &c.Resources
	}
	for i := range a.Ports {
		if a.Ports[i].ExternalPort == 0 {
			switch a.Ports[i].Type {
			case "http":
				a.Ports[i].ExternalPort = 80
			case "https":
				a.Ports[i].ExternalPort = 443
			}
		}
	}
}

func (a *App) Validate() (err error) {
	if len(a.Name) == 0 {
		return errors.New("deployment must have a name")
	}
	if a.Size == SizeUnknown {
		return errors.New("unknown size")
	}
	if err = a.Image.Validate(); err != nil {
		return err
	}
	if a.Resources != nil {
		if err = a.Resources.Validate(); err != nil {
			return err
		}
	}
	for _, port := range a.Ports {
		switch port.Type {
		case "http", "grpc", "tcp", "udp":
			continue
		default:
			return fmt.Errorf("invalid port type %q", port.Type)
		}
	}
	return nil
}

func (a *App) HasExternalPorts() bool {
	for _, p := range a.Ports {
		if p.ExternalPort != 0 {
			return true
		}
	}
	return false
}

type K8sManifest uint8

const (
	Kustomization K8sManifest = 1 + iota
	Deployment
	Service
	Secret
	ConfigMap
	Namespace
	HorizontalPodAutoscaler
)

func (m K8sManifest) Filename() string {
	switch m {
	case Deployment:
		return "deployment.yml"
	case Service:
		return "service.yml"
	case Secret:
		return "secret.yml"
	case ConfigMap:
		return "configmap.yml"
	case Namespace:
		return "namespace.yml"
	case Kustomization:
		return "kustomization.yml"
	case HorizontalPodAutoscaler:
		return "hpa.yml"
	default:
		panic(fmt.Sprintf("unknown k8s manifest: %d", m))
	}
}

const AppLabel = "app"

func (a *App) containerPorts() []corev1.ContainerPort {
	ports := make([]corev1.ContainerPort, len(a.Ports))
	for i, port := range a.Ports {
		ports[i] = corev1.ContainerPort{
			Name:          port.Type,
			ContainerPort: int32(port.Port),
		}
	}
	return ports
}

func (a *App) servicePorts() []corev1.ServicePort {
	ports := make([]corev1.ServicePort, len(a.Ports))
	for i, port := range a.Ports {
		ports[i] = corev1.ServicePort{
			Name:       port.Type,
			Port:       int32(port.ExternalPort),
			TargetPort: intstr.FromString(port.Type),
		}
		switch port.Type {
		case "http", "tcp", "grpc":
			ports[i].Protocol = corev1.ProtocolTCP
		case "udp":
			ports[i].Protocol = corev1.ProtocolUDP
		}
	}
	return ports
}

type KustomizeConfig struct {
	Resources    []string `json:"resources"`
	CommonLabels []string `json:"commonLabels,omitempty" yaml:"commonLabels"`
	Images       []struct {
		Name    string
		NewName string `json:"newName"`
		NewTag  string `json:"newTag"`
	} `json:"images,omitempty"`
	Patches            []any            `json:"patches,omitempty"`
	SecretGenerator    []map[string]any `json:"secretGenerator,omitempty"`
	ConfigMapGenerator []map[string]any `json:"configMapGenerator,omitempty"`
}

func (a *App) deployResources() corev1.ResourceRequirements {
	lim, _ := a.ResourceProfile.Limits(a.Size)
	req, _ := a.ResourceProfile.Requests(a.Size)
	return corev1.ResourceRequirements{
		Limits:   lim,
		Requests: req,
	}
}

func (a *App) deployment() *appsv1.Deployment {
	annotations := map[string]string{} // pod annotations
	for _, port := range a.Ports {
		if port.Type == "http" {
			annotations["prometheus.io/scrape"] = "true"
			annotations["prometheus.io/port"] = strconv.Itoa(int(port.ExternalPort))
			break
		}
	}
	if a.SkipPrometheus {
		annotations = nil
	}
	labels := make(map[string]string, len(a.ExtraLabels)+1)
	labels[AppLabel] = a.Name
	for k, v := range a.ExtraLabels {
		labels[k] = v
	}

	spec := appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{AppLabel: a.Name},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      labels,
				Annotations: annotations,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:            a.Name,
					Image:           a.Image.String(),
					ImagePullPolicy: corev1.PullAlways,
					Args:            a.Args,
					Ports:           a.containerPorts(),
					Resources:       a.deployResources(),
					EnvFrom: []corev1.EnvFromSource{
						{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: a.ConfigMapName,
								},
							},
						},
						{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: a.SecretName,
								},
							},
						},
					},
				}},
			},
		},
	}
	if a.Replicas > 0 {
		spec.Replicas = &a.Replicas
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              a.Name,
			Namespace:         a.Namespace,
			Labels:            labels,
			CreationTimestamp: metav1.Time{Time: time.Time{}},
		},
		Spec: spec,
	}
}

func (a *App) service() *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.Name,
			Namespace: a.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{AppLabel: a.Name},
			Ports:    a.servicePorts(),
		},
	}
}

func (a *App) configmap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.ConfigMapName,
			Namespace: a.Namespace,
		},
		Data: a.Config,
	}
}

func (a *App) secret() *corev1.Secret {
	data := make(map[string][]byte, len(a.Secrets))
	for k, v := range a.Secrets {
		data[k] = []byte(v)
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.SecretName,
			Namespace: a.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}
}

func (a *App) namespaceResource() *corev1.Namespace {
	if len(a.Namespace) == 0 {
		return nil
	}
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: a.Namespace,
		},
	}
}

func (a *App) hpa() *autoscale.HorizontalPodAutoscaler {
	if a.Scale == nil {
		return nil
	}

	metrics := make([]autoscale.MetricSpec, 0)
	if a.Scale.When.CPU > 0 {
		metrics = append(metrics, autoscale.MetricSpec{
			Type: autoscale.ResourceMetricSourceType,
			Resource: &autoscale.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscale.MetricTarget{
					Type:               autoscale.UtilizationMetricType,
					AverageUtilization: asPtr(int32(a.Scale.When.CPU * 100)),
				},
			},
		})
	}
	if a.Scale.When.Memory > 0 {
		metrics = append(metrics, autoscale.MetricSpec{
			Type: autoscale.ResourceMetricSourceType,
			Resource: &autoscale.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscale.MetricTarget{
					Type:               autoscale.UtilizationMetricType,
					AverageUtilization: asPtr(int32(a.Scale.When.Memory * 100)),
				},
			},
		})
	}

	return &autoscale.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "autoscaling/v2",
			Kind:       "HorizontalPodAutoscaler",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.Name,
			Namespace: a.Namespace,
		},
		Spec: autoscale.HorizontalPodAutoscalerSpec{
			MinReplicas: &a.Scale.From,
			MaxReplicas: a.Scale.To,
			ScaleTargetRef: autoscale.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       a.Name,
			},
			Metrics: metrics,
		},
	}
}

func (a *App) manifests() map[K8sManifest]any {
	m := map[K8sManifest]any{
		Service:    a.service(),
		ConfigMap:  a.configmap(),
		Secret:     a.secret(),
		Deployment: a.deployment(),
	}
	if len(a.Namespace) > 0 {
		m[Namespace] = a.namespaceResource()
	}
	if a.Scale != nil {
		m[HorizontalPodAutoscaler] = a.hpa()
	}
	return m
}

func (a *App) resources() []any {
	r := []any{
		a.secret(),
		a.configmap(),
		a.service(),
		a.deployment(),
	}
	if len(a.Namespace) > 0 {
		r = append(r, a.namespaceResource())
	}
	if a.Scale != nil {
		r = append(r, a.hpa())
	}
	return r
}

type defaultResourceProfile struct{}

func (d defaultResourceProfile) Limits(size Size) (corev1.ResourceList, error) {
	return d.doit(size, d.lim)
}

func (d defaultResourceProfile) Requests(size Size) (corev1.ResourceList, error) {
	return d.doit(size, d.req)
}

func (defaultResourceProfile) doit(size Size, fn func(Size) (string, int64)) (corev1.ResourceList, error) {
	mem, cpu := fn(size)
	return corev1.ResourceList{
		corev1.ResourceMemory: resource.MustParse(mem),
		corev1.ResourceCPU:    *resource.NewMilliQuantity(cpu, resource.BinarySI),
	}, nil
}

func (defaultResourceProfile) lim(size Size) (string, int64) {
	switch size {
	case SizeBig:
		return "512Mi", 250
	case SizeMed:
		return "256Mi", 100
	case SizeSml:
		return "128Mi", 50
	case SizeUnknown:
		fallthrough
	default:
		return "", 0
	}
}

func (defaultResourceProfile) req(size Size) (string, int64) {
	switch size {
	case SizeBig:
		return "256Mi", 100
	case SizeMed:
		return "128Mi", 50
	case SizeSml:
		return "64Mi", 10
	case SizeUnknown:
		fallthrough
	default:
		return "", 0
	}
}
