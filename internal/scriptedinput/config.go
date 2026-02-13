// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package scriptedinput

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"go.opentelemetry.io/collector/component"

	"github.com/splunk/tarunner/internal/conf"
)

const operatorType = "scripted_input"

func init() {
	operator.Register(operatorType, func() operator.Builder { return NewConfig() })
}

// NewConfig creates a new input config with default values
func NewConfig() *Config {
	return NewConfigWithID(operatorType)
}

// NewConfigWithID creates a new input config with default values
func NewConfigWithID(operatorID string) *Config {
	return &Config{
		InputConfig: helper.NewInputConfig(operatorID, operatorType),
	}
}

// Config is the configuration of a file input operator
type Config struct {
	BaseDir            string
	conf.Input         `mapstructure:",squash"`
	helper.InputConfig `mapstructure:",squash"`
}

// Build will build a file input operator from the supplied configuration
func (c Config) Build(set component.TelemetrySettings) (operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(set)
	if err != nil {
		return nil, err
	}

	input := &ScriptedInput{
		InputOperator: inputOperator,
		logger:        set.Logger,
		doneChan:      make(chan struct{}),
		cfg:           c,
	}

	return input, nil
}
