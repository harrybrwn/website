package main

import (
	"errors"
	"os"
	"testing"
	"text/template"

	"gopkg.in/yaml.v3"
)

func TestUnmarshal_JSONTextYAML(t *testing.T) {
	type Conf struct {
		Image Image
		Size  Size
	}
	type table struct {
		body string
		exp  Conf
	}
	var exp = Conf{Image: Image{Registry: "docker.io", Name: "username/image", Tag: "latest"}}
	for _, tt := range []table{
		{
			body: `
image:
  registry: docker.io
  name: username/image
  tag: latest
size: small`,
			exp: Conf{
				Size:  SizeSml,
				Image: exp.Image,
			},
		},
		{
			body: "image: docker.io/username/image:latest\nsize: b",
			exp: Conf{
				Size:  SizeBig,
				Image: exp.Image,
			},
		},
		{
			body: "image: docker.io/username/image:latest\nsize: medium",
			exp: Conf{
				Size:  SizeMed,
				Image: exp.Image,
			},
		},
	} {
		var c Conf
		err := yaml.Unmarshal([]byte(tt.body), &c)
		if err != nil {
			t.Fatal(err)
		}
		if c.Image.Name != tt.exp.Image.Name {
			t.Errorf("wrong image name: got %v, want %v", c.Image.Name, tt.exp.Image.Name)
		}
		if c.Image.Tag != tt.exp.Image.Tag {
			t.Errorf("wrong image tag: got %v, want %v", c.Image.Tag, tt.exp.Image.Tag)
		}
		if c.Image.Registry != tt.exp.Image.Registry {
			t.Errorf("wrong image registry: got %v, want %v", c.Image.Registry, tt.exp.Image.Registry)
		}
		if c.Size != tt.exp.Size {
			t.Errorf("wrong size: got %v, want %v", c.Size, tt.exp.Size)
		}
	}
}

func TestParseImage(t *testing.T) {
	type table struct {
		in  string
		exp Image
		err error
	}
	for _, tt := range []table{
		{in: "", err: errEmptyImageString},
		{in: "username/image", exp: Image{Name: "username/image"}},
		{in: "docker.io/username/image", exp: Image{Name: "username/image", Registry: "docker.io"}},
		{in: "docker.io/username/image:v0.1", exp: Image{Name: "username/image", Registry: "docker.io", Tag: "v0.1"}},
	} {
		var i Image
		err := parseImage(&i, tt.in)
		if err != nil {
			if tt.err == nil {
				t.Error(err)
			} else if !errors.Is(tt.err, err) {
				t.Errorf("wrong error: want %v, got %v", tt.err, err)
			}
			continue
		} else if tt.err != nil {
			t.Error("expected an error")
			continue
		}
		if i.Name != tt.exp.Name {
			t.Errorf("wrong name: got %v, want %v", i.Name, tt.exp.Name)
		}
		if i.Tag != tt.exp.Tag {
			t.Errorf("wrong Tag: got %v, want %v", i.Tag, tt.exp.Tag)
		}
		if i.Registry != tt.exp.Registry {
			t.Errorf("wrong Registry: got %v, want %v", i.Registry, tt.exp.Registry)
		}
	}
}

func TestTemplateGeneration(t *testing.T) {
	t.Skip()
	var (
		err    error
		config K8sGenConfig
	)
	tmpl := template.New("k8s").Funcs(templateFuncs)
	tmpl, err = tmpl.ParseFS(tmpls, "templates/*")
	if err != nil {
		t.Fatal(err)
	}
	app := App{
		//Env:  "prd",
		Name: "test-app",
		Image: Image{
			Name: "testing",
			Tag:  "1.1.1",
		},
		Ports: []Port{{
			Type:         "http",
			Port:         8080,
			ExternalPort: 80,
		}},
		Config: map[string]string{
			"PORT": "8080",
			"HOST": "0.0.0.0",
		},
		Secrets: map[string]string{
			"PASSWORD": "pa$$w0rd1",
			"API_KEY":  "buttbuttbuttx",
		},
	}
	if err = app.Validate(); err != nil {
		t.Fatal(err)
	}
	app.Calc(&config)
	out := os.Stdout
	err = tmpl.ExecuteTemplate(out, "deployment.yml.tmpl", &app)
	if err != nil {
		t.Fatal(err)
	}
	println()
	err = tmpl.ExecuteTemplate(out, "configmap.yml.tmpl", &app)
	if err != nil {
		t.Fatal(err)
	}
	println()
	err = tmpl.ExecuteTemplate(out, "secret.yml.tmpl", &app)
	if err != nil {
		t.Fatal(err)
	}
	// println()
	err = tmpl.ExecuteTemplate(out, "service.yml.tmpl", &app)
	if err != nil {
		t.Fatal(err)
	}
	err = tmpl.ExecuteTemplate(out, "kustomization.yml.tmpl", &app)
	if err != nil {
		t.Fatal(err)
	}
	err = tmpl.ExecuteTemplate(out, "kustomization.yml", &app)
	if err != nil {
		t.Fatal(err)
	}
	println()
}

func must[T any](a T, e error) T {
	if e != nil {
		panic(e)
	}
	return a
}
