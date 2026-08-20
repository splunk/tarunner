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

// watchingReceiver subscribes to one or more observers and starts a sub-receiver
// per directory endpoint.
type watchingReceiver struct {
	cfg      Config
	settings receiver.Settings
	logs     consumer.Logs
	opts     factoryOptions

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
		w.settings.Logger.Info("splunk_inputs: subscribing to observer", zap.String("observer", obsID.String()))
		ext, ok := host.GetExtensions()[obsID]
		if !ok {
			return fmt.Errorf("splunk_inputs: observer extension %q not found", obsID)
		}
		obs, ok := ext.(observer.Observable)
		if !ok {
			return fmt.Errorf("splunk_inputs: extension %q does not implement observer.Observable", obsID)
		}
		w.settings.Logger.Info("splunk_inputs: calling ListAndWatch", zap.String("observer", obsID.String()))
		obs.ListAndWatch(w)
		w.settings.Logger.Info("splunk_inputs: ListAndWatch returned", zap.String("observer", obsID.String()))
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
		inputs, err := tabuilder.ReadInputs(path)
		if err != nil {
			w.settings.Logger.Error("splunk_inputs: failed to read inputs for endpoint",
				zap.String("path", path), zap.Error(err))
			continue
		}
		transforms, err := tabuilder.ReadTransforms(path)
		if err != nil {
			w.settings.Logger.Error("splunk_inputs: failed to read transforms for endpoint",
				zap.String("path", path), zap.Error(err))
			continue
		}
		props, err := tabuilder.ReadProps(path)
		if err != nil {
			w.settings.Logger.Error("splunk_inputs: failed to read props for endpoint",
				zap.String("path", path), zap.Error(err))
			continue
		}
		receivers, err := w.opts.createReceivers(w.ctx, inputs, transforms, props, path, w.logs, w.settings)
		if err != nil {
			w.settings.Logger.Error("splunk_inputs: failed to create receivers for endpoint",
				zap.String("path", path), zap.Error(err))
			continue
		}
		r := packReceivers(receivers)
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
