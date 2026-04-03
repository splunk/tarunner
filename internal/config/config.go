// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"go.opentelemetry.io/collector/confmap"
	"go.yaml.in/yaml/v3"
)

type Config struct {
	Type     string `mapstructure:"type"`
	Endpoint string `mapstructure:"endpoint"`
	Token    string `mapstructure:"token"`
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rawConf map[string]any
	if err := yaml.Unmarshal(b, &rawConf); err != nil {
		return nil, err
	}
	c := confmap.NewFromStringMap(rawConf)
	cfg := &Config{
		Type:     "otlp_http",
		Endpoint: "http://localhost:4318",
	}
	err = c.Unmarshal(cfg)
	return cfg, err
}
