package main

import (
	"errors"
	"fmt"
	"log"
	"maps"
	"path/filepath"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscale "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
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
	ExtraLabels     map[string]string `json:"labels" yaml:"labels"`
	MetricsPath     string            `json:"metrics_path" yaml:"metrics_path"`
	Skip            bool
	ResourceProfile ResourceProfile `json:"-" yaml:"-"`
}

type Port struct {
	Type         string
	Port         uint16
	ExternalPort uint16 `yaml:"external_port"`
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

type Resources struct {
	Limits, Requests *ResourceSettings
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
		resources := ResourceSizeOrRaw{
			Sizes: c.Resources,
		}
		if a.Resources != nil {
			resources.LimitSettings = a.Resources.Limits
			resources.RequestSettings = a.Resources.Requests
		}
		a.ResourceProfile = &resources
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
	if a.Scale != nil {
		// it the upper and lower bounds are the same then we don't need an
		// autoscaler. also check the deployment replicas number.
		if a.Scale.From == a.Scale.To && (a.Replicas == 0 || a.Replicas == a.Scale.From) {
			a.Replicas = a.Scale.From
			a.Scale = nil
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

func (a *App) envFrom() []corev1.EnvFromSource {
	return []corev1.EnvFromSource{
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
	}
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
	ServiceMonitor
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
	case ServiceMonitor:
		return "servicemonitor.yml"
	default:
		panic(fmt.Sprintf("unknown k8s manifest: %d", m))
	}
}

const AppLabel = "app"

const (
	K8sLabelName      = "app.kubernetes.io/name"
	K8sLabelPartOf    = "app.kubernetes.io/part-of"
	K8sLabelManagedBy = "app.kubernetes.io/managed-by"
)

func (a *App) CheckExtraResources(dir string) error {
	for _, r := range a.ExtraResources {
		f := filepath.Join(dir, r)
		if !exists(f) {
			return fmt.Errorf("extra resource %q does not exist", f)
		}
	}
	return nil
}

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

// Depricated: use sigs.k8s.io/kustomize/api/types.Kustomization
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
	lim, err := a.ResourceProfile.Limits(a.Size)
	if err != nil {
		log.Printf("[warn]: %q", err)
	}
	req, err := a.ResourceProfile.Requests(a.Size)
	if err != nil {
		log.Printf("[warn]: %q", err)
	}
	return corev1.ResourceRequirements{
		Limits:   lim,
		Requests: req,
	}
}

func (a *App) commonLabels() map[string]string {
	return map[string]string{
		K8sLabelName:   a.Name,
		K8sLabelPartOf: a.Name,
		// K8sLabelManagedBy: "gopkgs.hrry.dev.lab",
	}
}

func (a *App) deployment() *appsv1.Deployment {
	annotations := map[string]string{} // pod annotations
	if a.SkipPrometheus {
		annotations = nil
	}
	labels := make(map[string]string, len(a.ExtraLabels)+4)
	labels[AppLabel] = a.Name
	maps.Copy(labels, a.commonLabels())
	maps.Copy(labels, a.ExtraLabels)

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
					EnvFrom:         a.envFrom(),
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
			Labels:            without(labels, AppLabel),
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
			Labels:    merge(a.commonLabels(), a.ExtraLabels),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{AppLabel: a.Name},
			Ports:    a.servicePorts(),
		},
	}
}

func (a *App) configmap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.ConfigMapName,
			Namespace: a.Namespace,
			Labels:    merge(a.commonLabels(), a.ExtraLabels),
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
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.SecretName,
			Namespace: a.Namespace,
			Labels:    merge(a.commonLabels(), a.ExtraLabels),
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
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   a.Namespace,
			Labels: a.commonLabels(),
		},
	}
}

func (a *App) hpa() *autoscale.HorizontalPodAutoscaler {
	if a.Scale == nil {
		return nil
	}
	if a.Scale.From == 1 && a.Scale.To == 1 {
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
			Labels:    merge(a.commonLabels(), a.ExtraLabels),
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
	if len(a.Namespace) > 0 && strings.ToLower(a.Namespace) != "default" {
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
	if len(a.Namespace) > 0 && strings.ToLower(a.Namespace) != "default" {
		r = append(r, a.namespaceResource())
	}
	if a.Scale != nil {
		r = append(r, a.hpa())
	}
	return r
}

type KV[V any] struct {
	Key string
	Val V
}

func without[T any](m map[string]T, k string) map[string]T {
	res := make(map[string]T, len(m))
	maps.Copy(res, m)
	delete(res, k)
	return res
}

func with[T any](m map[string]T, kvs ...KV[T]) map[string]T {
	for _, kv := range kvs {
		m[kv.Key] = kv.Val
	}
	return m
}

func merge[T any](m map[string]T, from map[string]T) map[string]T {
	maps.Copy(m, from)
	return m
}
