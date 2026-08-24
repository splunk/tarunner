// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
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

// resolveTADir returns the directory to read conf files from.
// When Path is set, it is used directly (single-TA mode).
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
