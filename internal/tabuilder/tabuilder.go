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
// dispatching by the stanza name's input kind. Stanzas with disabled=1 or
// unsupported kinds are skipped silently.
func CreateReceivers(ctx context.Context, inputs []conf.Input, transforms []conf.Transform, props []conf.Prop, baseDir string, next consumer.Logs, telemetrySettings component.TelemetrySettings) ([]receiver.Logs, error) {
	var receivers []receiver.Logs
	for _, input := range inputs {
		if input.Configuration.Stanza.IsDisabled() {
			continue
		}
		l, err := CreateReceiver(ctx, baseDir, next, input, transforms, props, telemetrySettings)
		if err != nil {
			return nil, fmt.Errorf("failed to create receiver %q: %w", input.Configuration.Stanza.Name, err)
		}
		if l == nil {
			continue
		}
		receivers = append(receivers, l)
	}
	return receivers, nil
}

// CreateReceiver builds a single logs receiver for an input stanza.
// Returns nil (no error) for unsupported input kinds.
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
		return f.CreateLogs(ctx, settings(f, parsed.Target, telemetrySettings), batchreceiver.Config{
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
		// unsupported kind — skip silently
		return nil, nil
	}
}

func settings(f receiver.Factory, path string, telemetrySettings component.TelemetrySettings) receiver.Settings {
	return receiver.Settings{
		ID:                component.MustNewIDWithName(f.Type().String(), path),
		TelemetrySettings: telemetrySettings,
	}
}

func confFilePaths(dirs []string, filename string) []string {
	paths := make([]string, len(dirs))
	for i, dir := range dirs {
		paths[i] = filepath.Join(dir, filename)
	}
	return paths
}

func splunkHomeDirs(splunkHome string) []string {
	etcDir := filepath.Join(splunkHome, "etc")

	appDirs, _ := filepath.Glob(filepath.Join(etcDir, "apps", "*"))
	sort.Strings(appDirs)

	dirs := []string{filepath.Join(etcDir, "system", "default")}
	for _, app := range appDirs {
		dirs = append(dirs, filepath.Join(app, "default"))
	}
	for _, app := range appDirs {
		dirs = append(dirs, filepath.Join(app, "local"))
	}
	dirs = append(dirs, filepath.Join(etcDir, "system", "local"))

	return dirs
}

// taDirs returns the conf search path for a single TA directory (default/ then local/).
func taDirs(taDir string) []string {
	return []string{
		filepath.Join(taDir, "default"),
		filepath.Join(taDir, "local"),
	}
}

// isSingleTA returns true when the path looks like a single TA directory
// (has a default/ or local/ subdirectory but no etc/apps/ layout).
func isSingleTA(path string) bool {
	_, errEtc := os.Stat(filepath.Join(path, "etc", "apps"))
	_, errDefault := os.Stat(filepath.Join(path, "default"))
	_, errLocal := os.Stat(filepath.Join(path, "local"))
	hasEtcApps := errEtc == nil
	hasTA := errDefault == nil || errLocal == nil
	return hasTA && !hasEtcApps
}

func confDirs(path string) []string {
	if isSingleTA(path) {
		return taDirs(path)
	}
	return splunkHomeDirs(path)
}

func readConfFiles(paths []string) ([][]byte, error) {
	var payloads [][]byte
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, b)
	}
	return payloads, nil
}

// ReadInputs discovers and merges inputs.conf files from dir using standard
// Splunk precedence. dir may be a Splunk home or a single TA directory.
// Returns nil (no error) when absent.
func ReadInputs(dir string) ([]conf.Input, error) {
	var layers [][]conf.Input
	for _, path := range confFilePaths(confDirs(dir), "inputs.conf") {
		b, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		appDir := filepath.Dir(filepath.Dir(path)) // strip /default or /local
		inputs, err := conf.ReadInput(b, appDir)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		layers = append(layers, inputs)
	}
	return conf.MergeInputs(layers), nil
}

// ReadTransforms discovers and merges transforms.conf files from dir using
// standard Splunk precedence. Returns nil (no error) when absent.
func ReadTransforms(dir string) ([]conf.Transform, error) {
	payloads, err := readConfFiles(confFilePaths(confDirs(dir), "transforms.conf"))
	if err != nil {
		return nil, err
	}
	var layers [][]conf.Transform
	for _, b := range payloads {
		transforms, err := conf.ReadTransforms(b)
		if err != nil {
			return nil, err
		}
		layers = append(layers, transforms)
	}
	return conf.MergeTransforms(layers), nil
}

// ReadProps discovers and merges props.conf files from dir using standard
// Splunk precedence. Returns nil (no error) when absent.
func ReadProps(dir string) ([]conf.Prop, error) {
	payloads, err := readConfFiles(confFilePaths(confDirs(dir), "props.conf"))
	if err != nil {
		return nil, err
	}
	var layers [][]conf.Prop
	for _, b := range payloads {
		props, err := conf.ReadProps(b)
		if err != nil {
			return nil, err
		}
		layers = append(layers, props)
	}
	return conf.MergeProps(layers), nil
}

// ReadOutputs merges outputs.conf across $SPLUNK_HOME using standard Splunk
// precedence. Use HTTPOut (or future TCPOut, etc.) to extract a specific type.
func ReadOutputs(splunkHome string) (conf.ConfMap, error) {
	payloads, err := readConfFiles(confFilePaths(confDirs(splunkHome), "outputs.conf"))
	if err != nil {
		return nil, err
	}
	return conf.ParseAndMergeConf(payloads)
}

func HTTPOut(merged conf.ConfMap) (*conf.Output, error) {
	return conf.HTTPOut(merged)
}

// CreateExporter builds a logs exporter from a merged outputs conf map.
func CreateExporter(merged conf.ConfMap, logger *zap.Logger, telemetrySettings component.TelemetrySettings) (exporter.Logs, error) {
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
