package tareceiver

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(component.MustNewType("ta"), createDefaultConfig, receiver.WithLogs(createLogsFunc, component.StabilityLevelDevelopment))
}

func createDefaultConfig() component.Config {
	return Config{}
}
