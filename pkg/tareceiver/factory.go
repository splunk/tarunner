// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(component.MustNewType("splunk_inputs"), createDefaultConfig, receiver.WithLogs(createLogsFunc, component.StabilityLevelDevelopment))
}

func createDefaultConfig() component.Config {
	return Config{}
}
