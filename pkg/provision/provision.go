package provision

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/imdario/mergo"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type Config struct {
	S3 S3Config `json:"s3" yaml:"s3" hcl:"s3,block"`
	DB DBConfig `json:"db" yaml:"db" hcl:"db,block"`
}

func (c *Config) ApplyFile(f io.Reader) error {
	var config Config
	err := json.NewDecoder(f).Decode(&config)
	if err != nil {
		return err
	}
	return mergo.Merge(c, &config, mergo.WithAppendSlice)
}

var (
	ConfigSchema = hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "s3"},
			{Type: "db"},
		},
	}
	// S3Schema hcl.BodySchema
	S3Schema, _ = gohcl.ImpliedBodySchema(&S3Config{})
	DBSchema, _ = gohcl.ImpliedBodySchema(&DBConfig{})
)

func (c *Config) HCLDecode(raw []byte, filename string) hcl.Diagnostics {
	f, diags := hclparse.NewParser().ParseHCL(raw, filename)
	if diags.HasErrors() {
		return diags
	}
	eval := hcl.EvalContext{
		Variables: map[string]cty.Value{},
		Functions: map[string]function.Function{},
	}
	diags = findVariables(&eval, &ConfigSchema, "", f.Body)
	if diags.HasErrors() {
		return diags
	}
	diags = gohcl.DecodeBody(f.Body, &eval, c)
	if diags.HasErrors() {
		return diags
	}
	return nil
}

func findVariables(eval *hcl.EvalContext, schema *hcl.BodySchema, prefix string, body hcl.Body) hcl.Diagnostics {
	content, diags := body.Content(schema)
	if diags.HasErrors() {
		return diags
	}
	for key, attr := range content.Attributes {
		var newPrefix string
		if prefix == "" {
			newPrefix = key
		} else {
			newPrefix = fmt.Sprintf("%s.%s", prefix, key)
		}
		fmt.Println("attr:", newPrefix, attr)
	}
	var diagnostics hcl.Diagnostics
	for _, block := range content.Blocks {
		var newPrefix string
		if prefix == "" {
			newPrefix = block.Type
		} else {
			newPrefix = fmt.Sprintf("%s.%s", prefix, block.Type)
		}
		fmt.Printf("block: %s => %+v\n", newPrefix, block)
		var sch *hcl.BodySchema
		switch newPrefix {
		case "s3":
			sch = S3Schema
		case "db":
			sch = DBSchema
		case "s3.user":
			sch, _ = gohcl.ImpliedBodySchema(&S3User{})
		default:
			return hcl.Diagnostics{
				{Severity: hcl.DiagError, Summary: "unknown block type"},
			}
		}
		diags = findVariables(eval, sch, newPrefix, block.Body)
		if diags.HasErrors() {
			diagnostics.Extend(diags)
		}
	}
	return diagnostics
}
