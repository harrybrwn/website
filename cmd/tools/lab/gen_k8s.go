package main

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	kustomize "sigs.k8s.io/kustomize/api/types"
)

//go:embed templates/*
var tmpls embed.FS

func NewGenK8sCmd() *cobra.Command {
	var (
		configFile = "./homelab.k8s.yml"
		destDir    = "./config/k8s/app"
		useK8sAPI  = false
		stdout     = false
		force      = false
		config     K8sGenConfig
	)
	c := cobra.Command{
		Use:   "k8s",
		Short: "Generate kubernetes manifests",

		PersistentPreRunE: func(*cobra.Command, []string) error {
			f, err := os.Open(configFile)
			if err != nil {
				return err
			}
			defer f.Close()
			err = yaml.NewDecoder(f).Decode(&config)
			if err != nil {
				return err
			}
			config.Calc()
			err = config.Validate()
			if err != nil {
				return err
			}
			return nil
		},

		RunE: func(cmd *cobra.Command, _ []string) error {
			g := K8sGenerator{
				config: config,
				dir:    destDir,
				force:  force,
			}
			if useK8sAPI {
				d := appsv1.Deployment{}
				res, err := yaml.Marshal(&d)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n", res)
				return errors.New("not yet implemented")
			} else if stdout {
				_, err := g.WriteTo(cmd.OutOrStdout())
				if err != nil {
					return err
				}
			} else {
				err := g.SaveTree()
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	f := c.Flags()
	f.StringVarP(&configFile, "config", "c", configFile, "file with values to generate manifests")
	f.StringVarP(&destDir, "destination", "d", destDir, "Directory that all the k8s manifests will be written to")
	f.BoolVar(&useK8sAPI, "use-k8s-api", useK8sAPI, "ues the kubernetes api to generate the applications")
	f.BoolVarP(&stdout, "stdout", "s", stdout, "write manifests to stdout instead of files")
	f.BoolVarP(&force, "force", "f", force, "force the existing manifests to be over written")
	return &c
}

type K8sGenerator struct {
	config K8sGenConfig
	dir    string
	force  bool
}

func (kg *K8sGenerator) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	defer enc.Close()
	enc.SetIndent(2)
	templs, err := templates()
	if err != nil {
		return 0, err
	}
	for _, app := range kg.config.Apps {
		if app.Skip {
			continue
		}
		for _, obj := range app.resources() {
			err := encodeYamlWithEncoder(enc, obj)
			if err != nil {
				return 0, err
			}
		}
		if !app.SkipPrometheus {
			_, err = buf.WriteString("---\n")
			if err != nil {
				return 0, err
			}
			err = templs.ExecuteTemplate(&buf, ServiceMonitor.Filename()+".tmpl", app)
			if err != nil {
				return 0, err
			}
		}
	}
	return io.Copy(w, &buf)
}

// This is super gross but its close to what the kubernetes api does so
// whatever.
func encodeYaml(o any, w io.Writer) error {
	enc := yaml.NewEncoder(w)
	defer enc.Close()
	enc.SetIndent(2)
	return encodeYamlWithEncoder(enc, o)
}

func encodeYamlWithEncoder(enc *yaml.Encoder, o any) error {
	json, err := json.Marshal(o)
	if err != nil {
		return err
	}
	var jsonObj map[string]any
	err = yaml.Unmarshal(json, &jsonObj)
	if err != nil {
		return err
	}
	delete(jsonObj, "status")
	deleteRec(jsonObj, "creationTimestamp")
	deleteRec(jsonObj, "strategy")
	return enc.Encode(jsonObj)
}

// deleteRec will delete a key from a map recursively.
func deleteRec(m map[string]any, key string) {
	delete(m, key)
	for _, v := range m {
		if obj, ok := v.(map[string]any); ok {
			deleteRec(obj, key)
		}
	}
}

func (kg *K8sGenerator) SaveTree() error {
	for name, app := range kg.config.Apps {
		if app.Skip {
			continue
		}
		err := kg.genTree(name, app)
		if err != nil {
			return err
		}
	}
	return nil
}

func (kg *K8sGenerator) genTree(name string, app *App) error {
	stdout := os.Stdout
	manifests := app.manifests()
	kustomize := &kustomize.Kustomization{Resources: app.ExtraResources[:]}
	dir := filepath.Join(kg.dir, name)
	err := os.Mkdir(dir, 0755)
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create app directory: %w", err)
		}
	}
	templs, err := templates()
	if err != nil {
		return err
	}

	if (app.Namespace == "" || app.Namespace == "default") && exists(filepath.Join(dir, Namespace.Filename())) {
		err = os.Remove(filepath.Join(dir, Namespace.Filename()))
		if err != nil {
			return err
		}
	}

	for n, manifest := range manifests {
		kustomize.Resources = append(kustomize.Resources, n.Filename())
		filename := filepath.Join(dir, n.Filename())
		if exists(filename) && !kg.force {
			fmt.Fprintf(stdout, "already exists: %q\n", filename)
			continue
		}
		file, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		err = encodeYaml(manifest, file)
		if err != nil {
			if e := file.Close(); e != nil {
				fmt.Fprintf(stdout, "error while closing file: %v", e)
			}
			return err
		}
		if err = file.Close(); err != nil {
			return err
		}
	}

	// Generate from templates
	templateManifests := []K8sManifest{}
	if !app.SkipPrometheus {
		templateManifests = append(templateManifests, ServiceMonitor)
	}
	for _, ktype := range templateManifests {
		filename := filepath.Join(dir, ktype.Filename())
		file, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		err = templs.ExecuteTemplate(file, ktype.Filename()+".tmpl", app)
		if err != nil {
			_ = file.Close()
			return err
		}
		if err = file.Close(); err != nil {
			return err
		}
		kustomize.Resources = append(kustomize.Resources, ktype.Filename())
	}
	return kg.createKustomization(stdout, dir, kustomize)
}

func (kg *K8sGenerator) createKustomization(stdout io.Writer, dir string, k *kustomize.Kustomization) error {
	filename := filepath.Join(dir, Kustomization.Filename())
	if exists(filename) && !kg.force {
		fmt.Fprintf(stdout, "already exists: %q\n", filename)
		return nil
	}
	file, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	sort.Strings(k.Resources)
	err = encodeYaml(k, file)
	if err != nil {
		return err
	}
	return nil
}

func templates() (*template.Template, error) {
	return template.New("").Funcs(templateFuncs).ParseFS(tmpls, "templates/*")
}

func exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

var templateFuncs = map[string]any{
	"b64": func(args ...any) string {
		var (
			err error
			b   strings.Builder
			enc = base64.StdEncoding
		)
		for _, arg := range args {
			switch a := arg.(type) {
			case string:
				_, err = b.WriteString(enc.EncodeToString([]byte(a)))
			case []byte:
				_, err = b.WriteString(enc.EncodeToString(a))
			default:
				continue
			}
			if err != nil {
				b.WriteString("<!error(\"")
				b.WriteString(err.Error())
				b.WriteString("\")>")
				continue
			}
		}
		return b.String()
	},
}

func asPtr[T any](v T) *T {
	return &v
}

func eat[T any](_ T, err error) error { return err }
