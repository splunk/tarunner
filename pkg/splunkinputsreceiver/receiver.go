// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

import (
	"context"
	"errors"
	"fmt"
	"os"

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

	if cfg.Path != "" && cfg.BaseDir != "" {
		return nil, fmt.Errorf("splunk_inputs: path and base_dir are mutually exclusive")
	}

	taDir, err := resolveTADir(cfg)
	if err != nil {
		return nil, err
	}

	inputs, err := tabuilder.ReadInputs(taDir)
	if err != nil {
		return nil, err
	}
	transforms, err := tabuilder.ReadTransforms(taDir)
	if err != nil {
		return nil, err
	}
	props, err := tabuilder.ReadProps(taDir)
	if err != nil {
		return nil, err
	}

	receivers, err := tabuilder.CreateReceivers(ctx, inputs, transforms, props, taDir, logs, settings.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return packReceivers(receivers), nil
}

// resolveTADir returns the directory to read conf files from.
// When Path is set, it is used directly (single-TA mode via receiver_creator).
// Otherwise base_dir or $SPLUNK_HOME is used for the full btool-style walk.
func resolveTADir(cfg Config) (string, error) {
	if cfg.Path != "" {
		return cfg.Path, nil
	}
	splunkHome := cfg.BaseDir
	if splunkHome == "" {
		splunkHome = os.Getenv("SPLUNK_HOME")
	}
	if splunkHome == "" {
		return "", fmt.Errorf("splunk_inputs: path is not set, base_dir is not set, and SPLUNK_HOME is not defined")
	}
	return splunkHome, nil
}
