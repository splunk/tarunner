// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer"

	"github.com/splunk/tarunner/internal/stanza"
	"github.com/splunk/tarunner/internal/tabuilder"
)

type factoryOptions struct {
	subReceivers map[string]SubReceiverFactory
}

func newFactoryOptions(opts ...Option) factoryOptions {
	options := factoryOptions{
		subReceivers: map[string]SubReceiverFactory{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return options
}

func (o factoryOptions) createLogsFunc(ctx context.Context, settings receiver.Settings, config component.Config, logs consumer.Logs) (receiver.Logs, error) {
	cfg := config.(Config)

	if len(cfg.WatchObservers) > 0 {
		if cfg.BaseDir != "" || cfg.Path != "" {
			return nil, fmt.Errorf("splunk_inputs: watch_observers is mutually exclusive with base_dir and path")
		}
		return &watchingReceiver{
			cfg:      cfg,
			settings: settings,
			logs:     logs,
			opts:     o,
			active:   make(map[observer.EndpointID]receiver.Logs),
		}, nil
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
	receivers, err := o.createReceivers(ctx, inputs, transforms, props, taDir, logs, settings)
	if err != nil {
		return nil, err
	}
	return packReceivers(receivers), nil
}

func (o factoryOptions) createReceivers(ctx context.Context, inputs []Input, transforms []Transform, props []Prop, baseDir string, next consumer.Logs, settings receiver.Settings) ([]receiver.Logs, error) {
	var receivers []receiver.Logs
	for _, input := range inputs {
		name := input.Configuration.Stanza.Name
		disabled := input.Configuration.Stanza.Params.Get("disabled")
		if disabled != nil && disabled.Value == "1" {
			settings.Logger.Info("splunk_inputs: skipping disabled stanza", zap.String("stanza", name))
			continue
		}
		settings.Logger.Info("splunk_inputs: creating receiver for stanza", zap.String("stanza", name))
		l, err := o.createReceiver(ctx, baseDir, next, input, transforms, props, settings)
		if err != nil {
			return nil, fmt.Errorf("failed to create receiver %q: %w", name, err)
		}
		if l == nil {
			settings.Logger.Info("splunk_inputs: skipping unsupported stanza kind", zap.String("stanza", name))
			continue
		}
		receivers = append(receivers, l)
	}
	return receivers, nil
}

func (o factoryOptions) createReceiver(ctx context.Context, baseDir string, next consumer.Logs, input Input, transforms []Transform, props []Prop, settings receiver.Settings) (receiver.Logs, error) {
	parsed, err := stanza.ParseName(input.Configuration.Stanza.Name)
	if err != nil {
		return nil, err
	}
	scheme := parsed.Kind
	if scheme == "" {
		scheme = "script"
	}
	if f, ok := o.subReceivers[strings.ToLower(scheme)]; ok {
		return f.CreateLogs(ctx, settings, ReceiverRequest{
			BaseDir:    baseDir,
			Path:       parsed.Target,
			Input:      input,
			Transforms: transforms,
			Props:      props,
		}, next)
	}
	return tabuilder.CreateReceiver(ctx, baseDir, next, input, transforms, props, settings.TelemetrySettings)
}

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
		w.settings.Logger.Info("splunk_inputs: building receivers for endpoint", zap.String("path", path))
		inputs, err := tabuilder.ReadInputs(path)
		if err != nil {
			w.settings.Logger.Error("splunk_inputs: failed to read inputs for endpoint",
				zap.String("path", path), zap.Error(err))
			continue
		}
		w.settings.Logger.Info("splunk_inputs: read inputs", zap.String("path", path), zap.Int("count", len(inputs)))
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
		w.settings.Logger.Info("splunk_inputs: created receivers for endpoint", zap.String("path", path), zap.Int("count", len(receivers)))
		r := packReceivers(receivers)
		if err := r.Start(w.ctx, w.host); err != nil {
			w.settings.Logger.Error("splunk_inputs: failed to start receiver for endpoint",
				zap.String("path", path), zap.Error(err))
			continue
		}
		w.settings.Logger.Info("splunk_inputs: started receiver for endpoint", zap.String("path", path))
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
