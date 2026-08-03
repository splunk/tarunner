// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkoutputsexporter

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"

	"github.com/splunk/tarunner/internal/outputsbuilder"
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

	baseDir := cfg.BaseDir
	if baseDir == "" {
		baseDir = os.Getenv("SPLUNK_HOME")
	}
	if baseDir == "" {
		return nil, fmt.Errorf("splunk_outputs: base_dir is not set and SPLUNK_HOME is not defined")
	}

	outputs, err := outputsbuilder.ReadOutputs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("splunk_outputs: failed to read outputs.conf from %q: %w", baseDir, err)
	}

	exporters, err := outputsbuilder.CreateExporters(outputs, settings.Logger, settings.TelemetrySettings)
	if err != nil {
		return nil, fmt.Errorf("splunk_outputs: %w", err)
	}

	return &aggregateExporter{exporters: exporters}, nil
}
