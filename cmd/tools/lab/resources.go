package main

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type ResourceProfile interface {
	Limits(size Size) (corev1.ResourceList, error)
	Requests(size Size) (corev1.ResourceList, error)
}

// ResourceSizeOrRaw will check that a raw resource spec can be used and will
// fallback to a size based resource configuration if one is not found.
type ResourceSizeOrRaw struct {
	Sizes           ResourceSizesConfig
	LimitSettings   *ResourceSettings
	RequestSettings *ResourceSettings
}

func (rsr *ResourceSizeOrRaw) Limits(size Size) (corev1.ResourceList, error) {
	if rsr.LimitSettings == nil {
		return rsr.Sizes.Limits(size)
	}
	return resourceSizesConfigResourceList(rsr.LimitSettings)
}

func (rsr *ResourceSizeOrRaw) Requests(size Size) (corev1.ResourceList, error) {
	if rsr.RequestSettings == nil {
		return rsr.Sizes.Requests(size)
	}
	return resourceSizesConfigResourceList(rsr.RequestSettings)
}

type ResourceSizesConfig map[string]*resourcesConfig

type resourcesConfig struct {
	Limits   *ResourceSettings `json:"limits" yaml:"limits"`
	Requests *ResourceSettings `json:"requests" yaml:"requests"`
}

type ResourceSettings struct {
	Memory string `json:"memory" yaml:"memory"`
	CPU    string `json:"cpu" yaml:"cpu"`
}

func (rs *ResourceSettings) Validate() error { return nil }

func (r ResourceSizesConfig) Limits(size Size) (corev1.ResourceList, error) {
	var dflts defaultResourceProfile
	c := r.conf(size)
	if c == nil || c.Limits == nil {
		return dflts.Limits(size)
	}
	return resourceSizesConfigResourceList(c.Limits)
}

func (r ResourceSizesConfig) Requests(size Size) (corev1.ResourceList, error) {
	var dflts defaultResourceProfile
	c := r.conf(size)
	if c == nil || c.Requests == nil {
		return dflts.Requests(size)
	}
	return resourceSizesConfigResourceList(c.Requests)
}

func resourceSizesConfigResourceList(conf *ResourceSettings) (corev1.ResourceList, error) {
	var err error
	res := make(corev1.ResourceList, 2)
	res[corev1.ResourceMemory], err = resource.ParseQuantity(conf.Memory)
	if err != nil {
		return nil, err
	}
	res[corev1.ResourceCPU], err = resource.ParseQuantity(conf.CPU)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r ResourceSizesConfig) conf(size Size) (c *resourcesConfig) {
	var ok bool
	switch size {
	case SizeBig:
		for _, k := range []string{"big", "large", "lg", "l", "b"} {
			c, ok = r[k]
			if ok {
				break
			}
		}
	case SizeMed:
		for _, k := range []string{"medium", "med", "m"} {
			c, ok = r[k]
			if ok {
				break
			}
		}
	case SizeSml:
		for _, k := range []string{"small", "sml", "sm", "s"} {
			c, ok = r[k]
			if ok {
				break
			}
		}
	}
	return c
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
