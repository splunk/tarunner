// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package scriptreceiver

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/noop"
	"go.opentelemetry.io/collector/component"

	"github.com/splunk/tarunner/internal/operator/prop"

	"github.com/splunk/tarunner/internal/scriptedinput"
	"github.com/splunk/tarunner/internal/scriptreceiver/internal/metadata"
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

func (t receiverType) InputConfig(config component.Config) operator.Config {
	rcfg := config.(*Config)
	oc := scriptedinput.NewConfig()
	oc.Input = rcfg.Input
	oc.BaseDir = rcfg.BaseDir

	index := "main"
	if indexParam := rcfg.Configuration.Stanza.Params.Get("index"); indexParam != nil {
		index = indexParam.Value
	}

	sourcetype := ""
	if sourceTypeParam := rcfg.Configuration.Stanza.Params.Get("sourcetype"); sourceTypeParam != nil {
		sourcetype = sourceTypeParam.Value
	}
	oc.Attributes = map[string]helper.ExprStringConfig{
		"com.splunk.index":      helper.ExprStringConfig(index),
		"com.splunk.sourcetype": helper.ExprStringConfig(sourcetype),
		"com.splunk.source":     helper.ExprStringConfig(rcfg.Configuration.Stanza.Name),
	}
	return operator.NewConfig(oc)
}
