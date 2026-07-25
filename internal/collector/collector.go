// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"github.com/splunk/tarunner/internal/config"
	"github.com/splunk/tarunner/internal/tabuilder"
)

// Run runs the collector with a baseDir working directory and an OTLP endpoint.
// The function returns an error if the collector could not start.
// The function returns a shutdown function handle if any work is scheduled,
// or nil if the TA has no activity and is therefore safe to exit.
func Run(baseDir string, cfg *config.Config) (func(), error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	meterProvider := noop.NewMeterProvider()
	tracerProvider := nooptrace.NewTracerProvider()
	var e exporter.Logs
	if cfg.Type == "otlp_http" {
		e, err = newOtlpHttpExporter(logger, cfg.Endpoint)
	} else {
		e, err = newHECExporter(logger, cfg.Endpoint, cfg.Token)
	}
	if err != nil {
		return nil, err
	}
	inputs, err := tabuilder.ReadInputs(baseDir)
	if err != nil {
		return nil, err
	}
	transforms, err := tabuilder.ReadTransforms(baseDir)
	if err != nil {
		return nil, err
	}
	props, err := tabuilder.ReadProps(baseDir)
	if err != nil {
		return nil, err
	}

	telemetrySettings := component.TelemetrySettings{
		Logger:         logger,
		MeterProvider:  meterProvider,
		TracerProvider: tracerProvider,
	}
	receivers, err := tabuilder.CreateReceivers(context.Background(), inputs, transforms, props, baseDir, e, telemetrySettings)
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
