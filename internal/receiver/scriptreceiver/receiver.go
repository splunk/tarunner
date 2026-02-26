// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package scriptreceiver

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/move"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/noop"
	"go.opentelemetry.io/collector/component"

	"github.com/splunk/tarunner/internal/operator/prop"

	"github.com/splunk/tarunner/internal/scriptedinput"
)

type scriptReceiver struct{}

// Type is the receiver type
func (scriptReceiver) Type() component.Type {
	return component.MustNewType("script")
}

// CreateDefaultConfig creates a config with type and version
func (scriptReceiver) CreateDefaultConfig() component.Config {
	return createDefaultConfig()
}

func createDefaultConfig() *Config {
	return &Config{}
}

// BaseConfig gets the base config from config, for now
func (scriptReceiver) BaseConfig(cfg component.Config) adapter.BaseConfig {
	rcfg := cfg.(*Config)
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

func (scriptReceiver) InputConfig(config component.Config) operator.Config {
	rcfg := config.(*Config)
	oc := scriptedinput.NewConfig()
	oc.Input = rcfg.Input
	oc.BaseDir = rcfg.BaseDir

	oc.Attributes = map[string]helper.ExprStringConfig{}

	if hostParam := rcfg.Configuration.Stanza.Params.Get("host"); hostParam != nil {
		// TODO: find a way to run host detection when requested.
		oc.Attributes["host"] = helper.ExprStringConfig(hostParam.Value)
	}

	if indexParam := rcfg.Configuration.Stanza.Params.Get("index"); indexParam != nil {
		oc.Attributes["index"] = helper.ExprStringConfig(indexParam.Value)
	}

	if sourceTypeParam := rcfg.Configuration.Stanza.Params.Get("sourcetype"); sourceTypeParam != nil {
		oc.Attributes["sourcetype"] = helper.ExprStringConfig(sourceTypeParam.Value)
	}

	if sourceParam := rcfg.Configuration.Stanza.Params.Get("source"); sourceParam != nil {
		oc.Attributes["source"] = helper.ExprStringConfig(sourceParam.Value)
	}

	return operator.NewConfig(oc)
}

func createSetSourceOperator() operator.Config {
	c := move.NewConfigWithID("start")
	c.From = entry.NewAttributeField("log.file.path")
	c.To = entry.NewAttributeField("source")
	c.OnError = "send_quiet"
	return operator.NewConfig(c)
}
