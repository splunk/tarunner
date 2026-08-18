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
	return CreateReceiverWithSettings(ctx, baseDir, next, input, transforms, props, receiver.Settings{
		TelemetrySettings: telemetrySettings,
	})
}

// CreateReceiverWithSettings builds a single logs receiver for an input stanza
// using the supplied receiver settings.
func CreateReceiverWithSettings(ctx context.Context, baseDir string, next consumer.Logs, input conf.Input, transforms []conf.Transform, props []conf.Prop, receiverSettings receiver.Settings) (receiver.Logs, error) {
	parsed, err := stanza.ParseName(input.Configuration.Stanza.Name)
	if err != nil {
		return nil, err
	}
	switch parsed.Kind {
	case "script", "":
		f := scriptreceiver.NewFactory()
		return f.CreateLogs(ctx, settings(f, parsed.Target, receiverSettings), &scriptreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		}, next)
	case "batch":
		f := batchreceiver.NewFactory()
		return f.CreateLogs(ctx, settings(f, parsed.Target, receiverSettings), batchreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		}, next)
	case "monitor":
		f := monitorreceiver.NewFactory()
		return f.CreateLogs(ctx, settings(f, parsed.Target, receiverSettings), monitorreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		}, next)
	case "wineventlog":
		f := wineventlogreceiver.NewFactory()
		return f.CreateLogs(ctx, settings(f, parsed.Target, receiverSettings), wineventlogreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		}, next)
	case "tcp":
		f := tcpreceiver.NewFactory()
		return f.CreateLogs(ctx, settings(f, parsed.Target, receiverSettings), tcpreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		}, next)
	case "udp":
		f := udpreceiver.NewFactory()
		return f.CreateLogs(ctx, settings(f, parsed.Target, receiverSettings), udpreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		}, next)
	default:
		return nil, fmt.Errorf("unsupported scheme %q", parsed.Kind)
	}
}

func settings(f receiver.Factory, path string, receiverSettings receiver.Settings) receiver.Settings {
	return receiver.Settings{
		ID:                component.MustNewIDWithName(f.Type().String(), path),
		TelemetrySettings: receiverSettings.TelemetrySettings,
		BuildInfo:         receiverSettings.BuildInfo,
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

// ReadOutputs reads outputs.conf from baseDir, preferring local/ over default/.
// Returns the [httpout] stanza, or nil if no [httpout] stanza is present.
func ReadOutputs(baseDir string) (*conf.Output, error) {
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

// CreateExporter builds a logs exporter from the [httpout] stanza.
func CreateExporter(output *conf.Output, logger *zap.Logger, telemetrySettings component.TelemetrySettings) (exporter.Logs, error) {
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
