package collector

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

func newExporter(logger *zap.Logger, endpoint string) (exporter.Logs, error) {
	f := otlphttpexporter.NewFactory()
	cfg := f.CreateDefaultConfig().(*otlphttpexporter.Config)
	cfg.ClientConfig.Endpoint = endpoint
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	e, err := f.CreateLogs(context.Background(), exporter.Settings{
		ID: component.MustNewID(f.Type().String()),
		TelemetrySettings: component.TelemetrySettings{
			Logger:         logger,
			TracerProvider: noop.NewTracerProvider(),
			MeterProvider:  metricnoop.NewMeterProvider(),
		},
	}, cfg)

	return e, err
}
