// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package monitorreceiver

import (
	"path/filepath"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/move"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/noop"

	"github.com/splunk/tarunner/internal/operator/prop"

	"github.com/splunk/tarunner/internal/script"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/file"
	"go.opentelemetry.io/collector/component"
)

type monitor struct{}

// Type is the receiver type
func (monitor) Type() component.Type {
	return component.MustNewType("monitor")
}

// CreateDefaultConfig creates a config with type and version
func (monitor) CreateDefaultConfig() component.Config {
	return createDefaultConfig()
}

func createDefaultConfig() *Config {
	return &Config{}
}

// BaseConfig gets the base config from config
func (monitor) BaseConfig(cfg component.Config) adapter.BaseConfig {
	rcfg := cfg.(Config)
	var operators []operator.Config
	operators = append(operators, createSetSourceOperator())

	for _, p := range rcfg.Props {
		ops := prop.CreateOperatorConfigs(p, rcfg.Transforms)
		operators = append(operators, ops...)
	}

	endNoop := noop.NewConfigWithID("end")
	operators = append(operators, operator.NewConfig(endNoop))

	return adapter.BaseConfig{
		Operators: operators,
	}
}

func createSetSourceOperator() operator.Config {
	c := move.NewConfigWithID("start")
	c.From = entry.NewAttributeField("log.file.path")
	c.To = entry.NewAttributeField("source")
	c.OnError = "send_quiet"
	return operator.NewConfig(c)
}

func (t monitor) InputConfig(config component.Config) operator.Config {
	rcfg := config.(Config)
	oc := file.NewConfig()
	path, err := script.DetermineCommandName(rcfg.BaseDir, rcfg.Input)
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
	oc.Attributes = map[string]helper.ExprStringConfig{}
	if hostParam := rcfg.Input.Configuration.Stanza.Params.Get("host"); hostParam != nil {
		// TODO: find a way to run host detection when requested.
		oc.Attributes["host"] = helper.ExprStringConfig(hostParam.Value)
	}

	if indexParam := rcfg.Input.Configuration.Stanza.Params.Get("index"); indexParam != nil {
		oc.Attributes["index"] = helper.ExprStringConfig(indexParam.Value)
	}

	if sourceTypeParam := rcfg.Input.Configuration.Stanza.Params.Get("sourcetype"); sourceTypeParam != nil {
		oc.Attributes["sourcetype"] = helper.ExprStringConfig(sourceTypeParam.Value)
	}

	if sourceParam := rcfg.Input.Configuration.Stanza.Params.Get("source"); sourceParam != nil {
		oc.Attributes["source"] = helper.ExprStringConfig(sourceParam.Value)
	}

	oc.IncludeFilePath = true
	oc.Encoding = "nop"

	return operator.NewConfig(oc)
}
