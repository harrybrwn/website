package provision

import (
	"encoding/json"
	"io"

	"github.com/imdario/mergo"
)

type Config struct {
	S3 S3Config `json:"s3" yaml:"s3"`
	DB DBConfig `json:"db" yaml:"db"`
}

func (c *Config) ApplyFile(f io.Reader) error {
	var config Config
	err := json.NewDecoder(f).Decode(&config)
	if err != nil {
		return err
	}
	return mergo.Merge(c, &config, mergo.WithAppendSlice, mergo.WithOverride)
}
