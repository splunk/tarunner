package scriptreceiver

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/parser/regex"
	"github.com/splunk/tarunner/internal/scriptedinput"
	"github.com/splunk/tarunner/internal/scriptreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
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
	oc := scriptedinput.NewConfig()
	oc.Input = rcfg.Input
	oc.BaseDir = rcfg.BaseDir

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
	return operator.NewConfig(oc)
}
