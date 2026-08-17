// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package tabuilder reads a technical addon's configuration files and builds
// the per-input logs receivers and per-output logs exporters. It is the single
// source of truth for the stanza -> component mapping, shared by the standalone
// collector runner (internal/collector) and the OTel plugins
// (pkg/splunkinputsreceiver, pkg/splunkoutputsexporter).
package tabuilder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/splunkhecexporter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/receiver"

	"github.com/splunk/tarunner/internal/conf"
	"github.com/splunk/tarunner/internal/receiver/batchreceiver"
	"github.com/splunk/tarunner/internal/receiver/monitorreceiver"
	"github.com/splunk/tarunner/internal/receiver/scriptreceiver"
	"github.com/splunk/tarunner/internal/receiver/tcpreceiver"
	"github.com/splunk/tarunner/internal/receiver/udpreceiver"
	"github.com/splunk/tarunner/internal/receiver/wineventlogreceiver"
	"github.com/splunk/tarunner/internal/stanza"
)

// CreateReceivers builds a logs receiver for every enabled input stanza,
// dispatching by the stanza name's input kind. Stanzas with disabled=1 are
// skipped.
func CreateReceivers(ctx context.Context, inputs []conf.Input, transforms []conf.Transform, props []conf.Prop, baseDir string, next consumer.Logs, telemetrySettings component.TelemetrySettings) ([]receiver.Logs, error) {
	var receivers []receiver.Logs
	for _, input := range inputs {
		disabled := input.Configuration.Stanza.Params.Get("disabled")
		if disabled != nil && disabled.Value == "1" {
			continue
		}
		l, err := CreateReceiver(ctx, baseDir, next, input, transforms, props, telemetrySettings)
		if err != nil {
			return nil, fmt.Errorf("failed to create receiver %q: %w", input.Configuration.Stanza.Name, err)
		}
		receivers = append(receivers, l)
	}
	return receivers, nil
}

// CreateReceiver builds a single logs receiver for an input stanza.
func CreateReceiver(ctx context.Context, baseDir string, next consumer.Logs, input conf.Input, transforms []conf.Transform, props []conf.Prop, telemetrySettings component.TelemetrySettings) (receiver.Logs, error) {
	parsed, err := stanza.ParseName(input.Configuration.Stanza.Name)
	if err != nil {
		return nil, err
	}
	switch parsed.Kind {
	case "script", "":
		f := scriptreceiver.NewFactory()
		return f.CreateLogs(ctx, settings(f, parsed.Target, telemetrySettings), &scriptreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		}, next)
	case "batch":
		f := batchreceiver.NewFactory()
		return f.CreateLogs(ctx, settings(f, parsed.Target, telemetrySettings), &scriptreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		}, next)
	case "monitor":
		f := monitorreceiver.NewFactory()
		return f.CreateLogs(ctx, settings(f, parsed.Target, telemetrySettings), monitorreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		}, next)
	case "wineventlog":
		f := wineventlogreceiver.NewFactory()
		return f.CreateLogs(ctx, settings(f, parsed.Target, telemetrySettings), wineventlogreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		}, next)
	case "tcp":
		f := tcpreceiver.NewFactory()
		return f.CreateLogs(ctx, settings(f, parsed.Target, telemetrySettings), tcpreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		}, next)
	case "udp":
		f := udpreceiver.NewFactory()
		return f.CreateLogs(ctx, settings(f, parsed.Target, telemetrySettings), udpreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		}, next)
	default:
		return nil, fmt.Errorf("unsupported scheme %q", parsed.Kind)
	}
}

func settings(f receiver.Factory, path string, telemetrySettings component.TelemetrySettings) receiver.Settings {
	return receiver.Settings{
		ID:                component.MustNewIDWithName(f.Type().String(), path),
		TelemetrySettings: telemetrySettings,
	}
}

// ReadInputs reads inputs.conf, preferring local/ over default/.
func ReadInputs(baseDir string) ([]conf.Input, error) {
	fileToRead := filepath.Join(baseDir, "local", "inputs.conf")
	if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
		fileToRead = filepath.Join(baseDir, "default", "inputs.conf")
		if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	b, err := os.ReadFile(fileToRead)
	if err != nil {
		return nil, err
	}
	return conf.ReadInput(b)
}

// ReadTransforms reads transforms.conf, preferring local/ over default/. It
// returns a nil slice (and no error) when the file is absent.
func ReadTransforms(baseDir string) ([]conf.Transform, error) {
	fileToRead := filepath.Join(baseDir, "local", "transforms.conf")
	if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
		fileToRead = filepath.Join(baseDir, "default", "transforms.conf")
		if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
	}
	b, err := os.ReadFile(fileToRead)
	if err != nil {
		return nil, err
	}
	return conf.ReadTransforms(b)
}

// ReadProps reads props.conf, preferring local/ over default/. It returns a nil
// slice (and no error) when the file is absent.
func ReadProps(baseDir string) ([]conf.Prop, error) {
	fileToRead := filepath.Join(baseDir, "local", "props.conf")
	if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
		fileToRead = filepath.Join(baseDir, "default", "props.conf")
		if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
	}
	b, err := os.ReadFile(fileToRead)
	if err != nil {
		return nil, err
	}
	return conf.ReadProps(b)
}

// ReadOutputs discovers and merges outputs.conf files from splunkHome using
// standard Splunk precedence (lowest to highest):
//
//  1. etc/system/default/outputs.conf
//  2. etc/apps/*/default/outputs.conf  (sorted by app name)
//  3. etc/apps/*/local/outputs.conf    (sorted by app name)
//  4. etc/system/local/outputs.conf
//
// Returns the merged stanza -> key -> value map. Callers use HTTPOut (or
// future TCPOut, SyslogOut, etc.) to extract a specific output type.
func ReadOutputs(splunkHome string) (map[string]map[string]string, error) {
	candidates := outputsConfPaths(splunkHome)

	var layers []map[string]map[string]string
	for _, path := range candidates {
		b, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		parsed, err := conf.ParseConf(b)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		layers = append(layers, parsed)
	}

	return conf.MergeConf(layers), nil
}

// HTTPOut extracts the [httpout] stanza from a merged outputs conf map.
// Returns conf.ErrNoHTTPOut if no [httpout] stanza is present.
func HTTPOut(merged map[string]map[string]string) (*conf.Output, error) {
	return conf.HTTPOut(merged)
}

// outputsConfPaths returns the ordered list of outputs.conf paths to consider,
// from lowest to highest precedence.
func outputsConfPaths(splunkHome string) []string {
	etcDir := filepath.Join(splunkHome, "etc")
	var paths []string

	// 1. system default
	paths = append(paths, filepath.Join(etcDir, "system", "default", "outputs.conf"))

	// 2. app defaults (sorted by app name)
	appDefaults, _ := filepath.Glob(filepath.Join(etcDir, "apps", "*", "default", "outputs.conf"))
	sort.Strings(appDefaults)
	paths = append(paths, appDefaults...)

	// 3. app locals (sorted by app name)
	appLocals, _ := filepath.Glob(filepath.Join(etcDir, "apps", "*", "local", "outputs.conf"))
	sort.Strings(appLocals)
	paths = append(paths, appLocals...)

	// 4. system local (highest precedence)
	paths = append(paths, filepath.Join(etcDir, "system", "local", "outputs.conf"))

	return paths
}

// CreateExporter builds a logs exporter from a merged outputs conf map.
// It extracts [httpout] and builds a HEC exporter. Future output types
// (tcpout, syslog, etc.) will be added here as additional cases.
func CreateExporter(merged map[string]map[string]string, logger *zap.Logger, telemetrySettings component.TelemetrySettings) (exporter.Logs, error) {
	output, err := conf.HTTPOut(merged)
	if err != nil {
		return nil, err
	}
	return newHECExporter(output, logger, telemetrySettings)
}

func newHECExporter(o *conf.Output, logger *zap.Logger, telemetrySettings component.TelemetrySettings) (exporter.Logs, error) {
	f := splunkhecexporter.NewFactory()
	cfg := f.CreateDefaultConfig().(*splunkhecexporter.Config)
	cfg.Endpoint = o.URI
	cfg.Token = configopaque.String(o.Token)
	cfg.TLS = configtls.ClientConfig{InsecureSkipVerify: true} // TODO: wire sslVerifyServerCert from outputs.conf
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	s := exporter.Settings{
		ID: component.MustNewID(f.Type().String()),
		TelemetrySettings: component.TelemetrySettings{
			Logger:         logger,
			TracerProvider: tracenoop.NewTracerProvider(),
			MeterProvider:  metricnoop.NewMeterProvider(),
		},
	}
	if telemetrySettings.Logger != nil {
		s.TelemetrySettings = telemetrySettings
	}
	return f.CreateLogs(context.Background(), s, cfg)
}
