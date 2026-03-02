// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"github.com/splunk/tarunner/internal/conf"
	"github.com/splunk/tarunner/internal/receiver/monitorreceiver"
	"github.com/splunk/tarunner/internal/receiver/scriptreceiver"
)

// Run runs the collector with a baseDir working directory and an OTLP endpoint.
// The function returns an error if the collector could not start.
// The function returns a shutdown function handle if any work is scheduled,
// or nil if the TA has no activity and is therefore safe to exit.
func Run(baseDir, endpoint string) (func(), error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	meterProvider := noop.NewMeterProvider()
	tracerProvider := nooptrace.NewTracerProvider()
	e, err := newExporter(logger, endpoint)
	if err != nil {
		return nil, err
	}
	apps, err := readApps(baseDir)
	if err != nil {
		return nil, err
	}
	var receivers []receiver.Logs
	for _, app := range apps {
		newReceivers, err := createReceivers(app.Name, app.Inputs, app.Transforms, app.Props, app.Dir, e, logger, meterProvider, tracerProvider)
		if err != nil {
			return nil, err
		}
		receivers = append(receivers, newReceivers...)

	}

	if len(receivers) == 0 {
		// No jobs to schedule. Exit.
		return nil, nil
	}

	h := host{}

	err = e.Start(context.Background(), h)
	if err != nil {
		return nil, err
	}
	for _, l := range receivers {
		if err = l.Start(context.Background(), h); err != nil {
			return nil, err
		}
	}

	shutDownFunc := func() {
		for _, l := range receivers {
			_ = l.Shutdown(context.Background())
		}
		_ = e.Shutdown(context.Background())
	}

	return shutDownFunc, nil
}

func createReceivers(name string, inputs []conf.Input, transforms []conf.Transform, props []conf.Prop, baseDir string, next consumer.Logs, logger *zap.Logger, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) ([]receiver.Logs, error) {
	var receivers []receiver.Logs
	for _, input := range inputs {
		disabled := input.Configuration.Stanza.Params.Get("disabled")
		if disabled != nil && disabled.Value == "1" {
			continue
		}
		l, err := createReceiver(name, baseDir, next, input, transforms, props, logger, meterProvider, tracerProvider)
		if err != nil {
			return nil, fmt.Errorf("failed to create receiver %q: %w", input.Configuration.Stanza.Name, err)
		}
		receivers = append(receivers, l)
	}
	return receivers, nil
}

func readApps(baseDir string) ([]conf.App, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	apps := make([]conf.App, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			inputs, err := readInputs(filepath.Join(baseDir, entry.Name()))
			if err != nil {
				return nil, err
			}
			transforms, err := readTransforms(filepath.Join(baseDir, entry.Name()))
			if err != nil {
				return nil, err
			}
			props, err := readProps(filepath.Join(baseDir, entry.Name()))
			if err != nil {
				return nil, err
			}
			apps = append(apps, conf.App{
				Name:       entry.Name(),
				Dir:        filepath.Join(baseDir, entry.Name()),
				Inputs:     inputs,
				Transforms: transforms,
				Props:      props,
			})
		}
	}
	return apps, nil
}

func readInputs(baseDir string) ([]conf.Input, error) {
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

func readTransforms(baseDir string) ([]conf.Transform, error) {
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

func readProps(baseDir string) ([]conf.Prop, error) {
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

func createReceiver(name, baseDir string, next consumer.Logs, input conf.Input, transforms []conf.Transform, props []conf.Prop, logger *zap.Logger, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) (receiver.Logs, error) {
	parsed, err := url.Parse(input.Configuration.Stanza.Name)
	if err != nil {
		return nil, err
	}
	switch parsed.Scheme {
	case "script", "":
		f := scriptreceiver.NewFactory()
		l, err := f.CreateLogs(context.Background(), receiver.Settings{
			ID: component.MustNewIDWithName(f.Type().String(), filepath.Join(name, parsed.Path)),
			TelemetrySettings: component.TelemetrySettings{
				Logger:         logger,
				MeterProvider:  meterProvider,
				TracerProvider: tracerProvider,
			},
		}, &scriptreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		},
			next)
		return l, err
	case "monitor":
		f := monitorreceiver.NewFactory()
		l, err := f.CreateLogs(context.Background(), receiver.Settings{
			ID: component.MustNewIDWithName(f.Type().String(), filepath.Join(name, parsed.Path)),
			TelemetrySettings: component.TelemetrySettings{
				Logger:         logger,
				MeterProvider:  meterProvider,
				TracerProvider: tracerProvider,
			},
		}, monitorreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		},
			next)
		return l, err
	default:
		return nil, fmt.Errorf("unknown scheme %q", parsed.Scheme)
	}
}
