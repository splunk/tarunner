// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkoutputsexporter

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"

	"github.com/splunk/tarunner/internal/tabuilder"
)

func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		component.MustNewType("splunk_outputs"),
		createDefaultConfig,
		exporter.WithLogs(createLogsFunc, component.StabilityLevelDevelopment),
	)
}

func createDefaultConfig() component.Config {
	return Config{}
}

func createLogsFunc(_ context.Context, settings exporter.Settings, config component.Config) (exporter.Logs, error) {
	cfg := config.(Config)

	splunkHome := cfg.BaseDir
	if splunkHome == "" {
		splunkHome = os.Getenv("SPLUNK_HOME")
	}
	if splunkHome == "" {
		return nil, fmt.Errorf("splunk_outputs: base_dir is not set and SPLUNK_HOME is not defined")
	}

	merged, err := tabuilder.ReadOutputs(splunkHome)
	if err != nil {
		return nil, fmt.Errorf("splunk_outputs: %w (base_dir: %s)", err, splunkHome)
	}

	exp, err := tabuilder.CreateExporter(merged, settings.Logger, settings.TelemetrySettings)
	if err != nil {
		return nil, fmt.Errorf("splunk_outputs: %w", err)
	}

	return exp, nil
}
