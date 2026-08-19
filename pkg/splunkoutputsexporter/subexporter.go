// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkoutputsexporter

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"

	"github.com/splunk/tarunner/internal/conf"
	"github.com/splunk/tarunner/internal/tabuilder"
)

type (
	Output = conf.Output
)

// ExporterRequest is passed to a sub-exporter factory for the [httpout]
// outputs.conf stanza.
type ExporterRequest struct {
	BaseDir string
	Output  Output
}

// SubExporterFactory creates a logs exporter for one outputs.conf stanza kind.
//
// Scheme returns the output stanza kind to match. Only the [httpout] stanza is
// read today. Schemes are normalized to lower-case before matching.
type SubExporterFactory interface {
	Scheme() string
	CreateLogs(context.Context, exporter.Settings, ExporterRequest) (exporter.Logs, error)
}

// Option configures the splunk_outputs factory.
type Option func(*factoryOptions)

// WithSubExporter registers a sub-exporter factory by Scheme. If another
// factory is already registered for the same scheme, it is replaced.
func WithSubExporter(f SubExporterFactory) Option {
	return func(o *factoryOptions) {
		if f == nil {
			return
		}
		o.subExporters[strings.ToLower(f.Scheme())] = f
	}
}

type factoryOptions struct {
	subExporters map[string]SubExporterFactory
}

func newFactoryOptions(opts ...Option) factoryOptions {
	options := factoryOptions{
		subExporters: map[string]SubExporterFactory{},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return options
}

func (o factoryOptions) createLogsFunc(ctx context.Context, settings exporter.Settings, config component.Config) (exporter.Logs, error) {
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
		return nil, fmt.Errorf("splunk_outputs: %w (base_dir: %s)", err, baseDir)
	}

	exp, err := o.createExporter(ctx, baseDir, *output, settings)
	if err != nil {
		return nil, fmt.Errorf("splunk_outputs: %w", err)
	}
	return exp, nil
}

func (o factoryOptions) createExporter(ctx context.Context, baseDir string, output Output, settings exporter.Settings) (exporter.Logs, error) {
	scheme := "httpout"
	if f, ok := o.subExporters[scheme]; ok {
		return f.CreateLogs(ctx, settings, ExporterRequest{
			BaseDir: baseDir,
			Output:  output,
		})
	}
	return tabuilder.CreateExporter(&output, settings.Logger, settings.TelemetrySettings)
}
