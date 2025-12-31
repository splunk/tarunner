package scriptedinput

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"github.com/splunk/tarunner/internal/conf"
	"go.opentelemetry.io/collector/component"
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
	helper.InputConfig `mapstructure:",squash"`
	conf.Input         `mapstructure:",squash"`
	BaseDir            string
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
