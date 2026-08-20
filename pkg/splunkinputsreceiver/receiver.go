// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer"

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

	exclusiveCount := 0
	if cfg.BaseDir != "" {
		exclusiveCount++
	}
	if cfg.Path != "" {
		exclusiveCount++
	}
	if len(cfg.WatchObservers) > 0 {
		exclusiveCount++
	}
	if exclusiveCount > 1 {
		return nil, fmt.Errorf("splunk_inputs: base_dir, path, and watch_observers are mutually exclusive")
	}

	if len(cfg.WatchObservers) > 0 {
		return &watchingReceiver{
			cfg:      cfg,
			settings: settings,
			logs:     logs,
			active:   make(map[observer.EndpointID]receiver.Logs),
		}, nil
	}

	taDir, err := resolveTADir(cfg)
	if err != nil {
		return nil, err
	}
	return buildReceiver(ctx, taDir, logs, settings)
}

// watchingReceiver subscribes to one or more observers and starts a sub-receiver
// per directory endpoint.
type watchingReceiver struct {
	cfg      Config
	settings receiver.Settings
	logs     consumer.Logs

	mu     sync.Mutex
	active map[observer.EndpointID]receiver.Logs
	host   component.Host
	ctx    context.Context
	cancel context.CancelFunc
}

var _ observer.Notify = (*watchingReceiver)(nil)

func (w *watchingReceiver) ID() observer.NotifyID {
	return observer.NotifyID(w.settings.ID.String())
}

func (w *watchingReceiver) Start(ctx context.Context, host component.Host) error {
	w.host = host
	w.ctx, w.cancel = context.WithCancel(ctx)

	for _, obsID := range w.cfg.WatchObservers {
		ext, ok := host.GetExtensions()[obsID]
		if !ok {
			return fmt.Errorf("splunk_inputs: observer extension %q not found", obsID)
		}
		obs, ok := ext.(observer.Observable)
		if !ok {
			return fmt.Errorf("splunk_inputs: extension %q does not implement observer.Observable", obsID)
		}
		obs.ListAndWatch(w)
	}
	return nil
}

func (w *watchingReceiver) Shutdown(ctx context.Context) error {
	if w.cancel != nil {
		w.cancel()
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	var errs []error
	for _, r := range w.active {
		errs = append(errs, r.Shutdown(ctx))
	}
	return errors.Join(errs...)
}

func (w *watchingReceiver) OnAdd(endpoints []observer.Endpoint) {
	for _, e := range endpoints {
		path, ok := e.Details.Env()["path"].(string)
		if !ok || path == "" {
			w.settings.Logger.Warn("splunk_inputs: endpoint missing path attribute", zap.String("id", string(e.ID)))
			continue
		}
		r, err := buildReceiver(w.ctx, path, w.logs, w.settings)
		if err != nil {
			w.settings.Logger.Error("splunk_inputs: failed to build receiver for endpoint",
				zap.String("path", path), zap.Error(err))
			continue
		}
		if err := r.Start(w.ctx, w.host); err != nil {
			w.settings.Logger.Error("splunk_inputs: failed to start receiver for endpoint",
				zap.String("path", path), zap.Error(err))
			continue
		}
		w.mu.Lock()
		w.active[e.ID] = r
		w.mu.Unlock()
	}
}

func (w *watchingReceiver) OnRemove(endpoints []observer.Endpoint) {
	for _, e := range endpoints {
		w.mu.Lock()
		r, ok := w.active[e.ID]
		if ok {
			delete(w.active, e.ID)
		}
		w.mu.Unlock()
		if ok {
			if err := r.Shutdown(w.ctx); err != nil {
				w.settings.Logger.Error("splunk_inputs: failed to shutdown receiver for removed endpoint",
					zap.String("id", string(e.ID)), zap.Error(err))
			}
		}
	}
}

func (w *watchingReceiver) OnChange(endpoints []observer.Endpoint) {
	w.OnRemove(endpoints)
	w.OnAdd(endpoints)
}

func buildReceiver(ctx context.Context, taDir string, logs consumer.Logs, settings receiver.Settings) (receiver.Logs, error) {
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
