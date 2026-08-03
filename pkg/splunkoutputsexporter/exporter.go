// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkoutputsexporter

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
)

type aggregateExporter struct {
	exporters []exporter.Logs
}

func (a *aggregateExporter) Start(ctx context.Context, host component.Host) error {
	var errs []error
	for _, e := range a.exporters {
		errs = append(errs, e.Start(ctx, host))
	}
	return errors.Join(errs...)
}

func (a *aggregateExporter) Shutdown(ctx context.Context) error {
	var errs []error
	for _, e := range a.exporters {
		errs = append(errs, e.Shutdown(ctx))
	}
	return errors.Join(errs...)
}

func (a *aggregateExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (a *aggregateExporter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	var errs []error
	for _, e := range a.exporters {
		errs = append(errs, e.ConsumeLogs(ctx, ld))
	}
	return errors.Join(errs...)
}
