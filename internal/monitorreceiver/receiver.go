// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package monitorreceiver

import (
	"net/url"
	"path/filepath"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/file"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/parser/regex"
	"go.opentelemetry.io/collector/component"

	"github.com/splunk/tarunner/internal/conf"
	"github.com/splunk/tarunner/internal/monitorreceiver/internal/metadata"
)

type receiverType struct{}

// Type is the receiver type
func (receiverType) Type() component.Type {
	return metadata.Type
}

// CreateDefaultConfig creates a config with type and version
func (receiverType) CreateDefaultConfig() component.Config {
	return createDefaultConfig()
}

func createDefaultConfig() *Config {
	return &Config{}
}

// BaseConfig gets the base config from config, for now
func (receiverType) BaseConfig(cfg component.Config) adapter.BaseConfig {
	rcfg := cfg.(*Config)
	var operators []operator.Config
	if rcfg.Transform != nil {
		r := regex.NewConfig()
		r.Regex = rcfg.Transform.Regex
		operators = append(operators, operator.NewConfig(r))
		if rcfg.Transform.Format != "" {
			panic("not supported yet")
		}
	}

	return adapter.BaseConfig{
		Operators: operators,
	}
}

func (t receiverType) InputConfig(config component.Config) operator.Config {
	rcfg := config.(*Config)
	oc := file.NewConfig()
	path, err := getPath(rcfg.BaseDir, rcfg.Input)
	if err != nil {
		return operator.NewConfig(oc)
	}
	allowlist := path
	if w := rcfg.Input.Configuration.Stanza.Params.Get("whitelist"); w != nil {
		allowlist = filepath.Join(path, w.Value)
	}
	oc.Include = []string{allowlist}
	if b := rcfg.Input.Configuration.Stanza.Params.Get("blacklist"); b != nil {
		oc.Exclude = []string{filepath.Join(path, b.Value)}
	}
	index := "main"
	if indexParam := rcfg.Input.Configuration.Stanza.Params.Get("index"); indexParam != nil {
		index = indexParam.Value
	}

	sourcetype := ""
	if sourceTypeParam := rcfg.Input.Configuration.Stanza.Params.Get("sourcetype"); sourceTypeParam != nil {
		sourcetype = sourceTypeParam.Value
	}
	oc.Attributes = map[string]helper.ExprStringConfig{
		"com.splunk.index":      helper.ExprStringConfig(index),
		"com.splunk.sourcetype": helper.ExprStringConfig(sourcetype),
		"com.splunk.source":     helper.ExprStringConfig(rcfg.Input.Configuration.Stanza.Name),
	}
	rcfg.Input.Configuration.Stanza.Params.Get("sourcetype")
	return operator.NewConfig(oc)
}

func getPath(baseDir string, input conf.Input) (string, error) {
	parsed, err := url.Parse(input.Configuration.Stanza.Name)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(parsed.Path) {
		return parsed.Path, nil
	} else {
		return filepath.Join(baseDir, parsed.Path), nil
	}
}
