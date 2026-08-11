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

	baseDir := cfg.BaseDir
	if baseDir == "" {
		baseDir = os.Getenv("SPLUNK_HOME")
	}
	if baseDir == "" {
		return nil, fmt.Errorf("splunk_outputs: path is not set and SPLUNK_HOME is not defined")
	}

	output, err := tabuilder.ReadOutputs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("splunk_outputs: failed to read outputs.conf from %q: %w", baseDir, err)
	}

	exp, err := tabuilder.CreateExporter(output, settings.Logger, settings.TelemetrySettings)
	if err != nil {
		return nil, fmt.Errorf("splunk_outputs: %w", err)
	}

	return exp, nil
}
