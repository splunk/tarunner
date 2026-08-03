// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package outputsbuilder reads outputs.conf and builds the per-stanza log
// exporters. It is the single source of truth for the outputs stanza ->
// exporter mapping, shared by the standalone collector runner and the
// splunk_outputs exporter.
package outputsbuilder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/splunkhecexporter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/exporter"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"github.com/splunk/tarunner/internal/conf"
)

// ReadOutputs reads outputs.conf from baseDir, preferring local/ over default/.
func ReadOutputs(baseDir string) ([]conf.Output, error) {
	fileToRead := filepath.Join(baseDir, "local", "outputs.conf")
	if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
		fileToRead = filepath.Join(baseDir, "default", "outputs.conf")
		if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	b, err := os.ReadFile(fileToRead)
	if err != nil {
		return nil, err
	}
	return conf.ReadOutputs(b)
}

// CreateExporters builds a logs exporter for every httpout stanza in outputs.
// Returns an error if no httpout stanzas are found.
func CreateExporters(outputs []conf.Output, logger *zap.Logger, telemetrySettings component.TelemetrySettings) ([]exporter.Logs, error) {
	var exporters []exporter.Logs
	for _, o := range outputs {
		if !o.IsHTTPOut() {
			continue
		}
		e, err := newHECExporter(o, logger, telemetrySettings)
		if err != nil {
			return nil, fmt.Errorf("failed to create exporter for stanza %q: %w", o.Name, err)
		}
		exporters = append(exporters, e)
	}
	if len(exporters) == 0 {
		return nil, errors.New("no httpout stanzas found in outputs.conf")
	}
	return exporters, nil
}

func newHECExporter(o conf.Output, logger *zap.Logger, telemetrySettings component.TelemetrySettings) (exporter.Logs, error) {
	f := splunkhecexporter.NewFactory()
	cfg := f.CreateDefaultConfig().(*splunkhecexporter.Config)
	cfg.Endpoint = o.URI
	cfg.Token = configopaque.String(o.Token)
	if o.BatchSize > 0 {
		cfg.MaxContentLengthLogs = uint(o.BatchSize)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	settings := exporter.Settings{
		ID: component.MustNewIDWithName(f.Type().String(), o.Name),
		TelemetrySettings: component.TelemetrySettings{
			Logger:         logger,
			TracerProvider: noop.NewTracerProvider(),
			MeterProvider:  metricnoop.NewMeterProvider(),
		},
	}
	if telemetrySettings.Logger != nil {
		settings.TelemetrySettings = telemetrySettings
	}
	return f.CreateLogs(context.Background(), settings, cfg)
}
