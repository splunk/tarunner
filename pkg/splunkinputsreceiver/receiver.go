// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

import (
	"context"
	"errors"
	"os"
	"path/filepath"

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

// resolveSplunkHome derives the Splunk installation root from a TA path.
// For a standard install the TA sits at $SPLUNK_HOME/etc/apps/<TA>, so the
// home is three levels up. If the derived path does not exist, $SPLUNK_HOME
// is used as a fallback.
func resolveSplunkHome(taPath string) string {
	derived := filepath.Dir(filepath.Dir(filepath.Dir(taPath)))
	if _, err := os.Stat(derived); err == nil {
		return derived
	}
	return os.Getenv("SPLUNK_HOME")
}
