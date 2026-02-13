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
	"github.com/splunk/tarunner/internal/monitorreceiver"
	"github.com/splunk/tarunner/internal/scriptreceiver"
)

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
	inputs, err := readInputs(baseDir)
	if err != nil {
		return nil, err
	}
	transforms, err := readTransforms(baseDir)
	if err != nil {
		return nil, err
	}

	receivers, err := createReceivers(inputs, transforms, baseDir, e, logger, meterProvider, tracerProvider)
	if err != nil {
		return nil, err
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

func createReceivers(inputs []conf.Input, transforms []conf.Transform, baseDir string, next consumer.Logs, logger *zap.Logger, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) ([]receiver.Logs, error) {
	var receivers []receiver.Logs
	for _, input := range inputs {
		disabled := input.Configuration.Stanza.Params.Get("disabled")
		if disabled != nil && disabled.Value == "1" {
			continue
		}
		var transform *conf.Transform
		if sourceType := input.Configuration.Stanza.Params.Get("sourcetype"); sourceType != nil {
			for _, t := range transforms {
				if t.Name == sourceType.Value {
					transform = &t
				}
			}
		}
		l, err := createReceiver(baseDir, next, input, transform, logger, meterProvider, tracerProvider)
		if err != nil {
			return nil, err
		}
		receivers = append(receivers, l)
	}
	return receivers, nil
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

func createReceiver(baseDir string, next consumer.Logs, input conf.Input, transform *conf.Transform, logger *zap.Logger, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) (receiver.Logs, error) {
	parsed, err := url.Parse(input.Configuration.Stanza.Name)
	if err != nil {
		return nil, err
	}
	switch parsed.Scheme {
	case "script", "":
		f := scriptreceiver.NewFactory()
		l, err := f.CreateLogs(context.Background(), receiver.Settings{
			ID: component.MustNewIDWithName(f.Type().String(), parsed.Path),
			TelemetrySettings: component.TelemetrySettings{
				Logger:         logger,
				MeterProvider:  meterProvider,
				TracerProvider: tracerProvider,
			},
		}, &scriptreceiver.Config{
			Input:   input,
			BaseDir: baseDir,
		},
			next)
		return l, err
	case "monitor":
		f := monitorreceiver.NewFactory()
		l, err := f.CreateLogs(context.Background(), receiver.Settings{
			ID: component.MustNewIDWithName(f.Type().String(), parsed.Path),
			TelemetrySettings: component.TelemetrySettings{
				Logger:         logger,
				MeterProvider:  meterProvider,
				TracerProvider: tracerProvider,
			},
		}, monitorreceiver.Config{
			Input:   input,
			BaseDir: baseDir,
		},
			next)
		return l, err
	default:
		return nil, fmt.Errorf("unknown scheme %q", parsed.Scheme)
	}
}
