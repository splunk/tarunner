// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/splunk/tarunner/internal/tabuilder"
)

var nopInstance = &nopReceiver{}

type nopReceiver struct {
	component.StartFunc
	component.ShutdownFunc
}

type aggregateReceiver struct {
	receivers []receiver.Logs
}

func (a aggregateReceiver) Start(ctx context.Context, host component.Host) error {
	var errs []error
	for _, r := range a.receivers {
		errs = append(errs, r.Start(ctx, host))
	}
	return errors.Join(errs...)
}

func (a aggregateReceiver) Shutdown(ctx context.Context) error {
	var errs []error
	for _, r := range a.receivers {
		errs = append(errs, r.Shutdown(ctx))
	}
	return errors.Join(errs...)
}

func packReceivers(receivers []receiver.Logs) receiver.Logs {
	switch len(receivers) {
	case 0:
		return nopInstance
	case 1:
		return receivers[0]
	default:
		return aggregateReceiver{
			receivers: receivers,
		}
	}
}

func createLogsFunc(ctx context.Context, settings receiver.Settings, config component.Config, logs consumer.Logs) (receiver.Logs, error) {
	cfg := config.(Config)
	baseDir := cfg.BaseDir
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

	receivers, err := tabuilder.CreateReceivers(ctx, inputs, transforms, props, baseDir, logs, settings.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return packReceivers(receivers), nil
}
