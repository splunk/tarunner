// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/splunk/tarunner/internal/tabuilder"
)

const debounceDuration = 500 * time.Millisecond

type splunkInputsReceiver struct {
	splunkHome string
	handler    *observerHandler
	watcher    *fsnotify.Watcher
	doneCh     chan struct{}
}

func newSplunkInputsReceiver(splunkHome string, options factoryOptions, settings receiver.Settings, next consumer.Logs) *splunkInputsReceiver {
	return &splunkInputsReceiver{
		splunkHome: splunkHome,
		handler:    newObserverHandler(splunkHome, options, settings, next),
		doneCh:     make(chan struct{}),
	}
}

func (r *splunkInputsReceiver) Start(ctx context.Context, host component.Host) error {
	r.handler.host = host
	taDirs, err := tabuilder.DiscoverTAs(r.splunkHome)
	if err != nil {
		return err
	}
	if err := r.handler.OnAdd(ctx, taDirs); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	r.watcher = watcher

	for _, dir := range tabuilder.WatchDirs(r.splunkHome) {
		_ = watcher.Add(dir) // best-effort; dirs may not exist yet
	}
	for _, taDir := range taDirs {
		_ = watcher.Add(filepath.Join(taDir, "default"))
		_ = watcher.Add(filepath.Join(taDir, "local"))
	}

	go r.watchLoop(ctx)
	return nil
}

func (r *splunkInputsReceiver) Shutdown(ctx context.Context) error {
	if r.watcher != nil {
		_ = r.watcher.Close()
		<-r.doneCh
	}
	r.handler.shutdown(ctx)
	return nil
}

func (r *splunkInputsReceiver) watchLoop(ctx context.Context) {
	defer close(r.doneCh)

	logger := r.handler.settings.Logger

	// pending tracks which TA dirs need reconciling, keyed by taDir.
	// system-layer changes use the empty string as a sentinel for "all TAs".
	pending := map[string]struct{}{}
	var debounce <-chan time.Time

	appsDir := filepath.Join(r.splunkHome, "etc", "apps")
	systemDir := filepath.Join(r.splunkHome, "etc", "system")

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}
			switch {
			case strings.HasPrefix(event.Name, systemDir):
				// system conf change affects all TAs — use sentinel
				pending[""] = struct{}{}
			case strings.HasPrefix(event.Name, appsDir):
				rel, _ := filepath.Rel(appsDir, event.Name)
				if !strings.Contains(rel, string(filepath.Separator)) {
					// direct child of appsDir — TA directory added or removed
					pending[""] = struct{}{}
				} else {
					// event inside a TA's default/ or local/ — target just that TA
					pending[taDirFromPath(event.Name, appsDir)] = struct{}{}
				}
			}
			debounce = time.After(debounceDuration)
		case err, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
			logger.Warn("splunk_inputs: watcher error", zap.Error(err))
		case <-debounce:
			debounce = nil
			r.reconcile(ctx, pending)
			pending = map[string]struct{}{}
		}
	}
}

// taDirFromPath returns the TA root directory from an event path under appsDir,
// or empty string if the event is directly in appsDir (TA added/removed).
func taDirFromPath(eventPath, appsDir string) string {
	rel, err := filepath.Rel(appsDir, eventPath)
	if err != nil {
		return ""
	}
	parts := strings.SplitN(rel, string(filepath.Separator), 2)
	if parts[0] == "" || parts[0] == "." {
		return ""
	}
	return filepath.Join(appsDir, parts[0])
}

func (r *splunkInputsReceiver) reconcile(ctx context.Context, pending map[string]struct{}) {
	logger := r.handler.settings.Logger

	current, err := tabuilder.DiscoverTAs(r.splunkHome)
	if err != nil {
		logger.Error("splunk_inputs: failed to discover TAs", zap.Error(err))
		return
	}

	desired := make(map[string]struct{}, len(current))
	for _, d := range current {
		desired[d] = struct{}{}
	}

	_, allTAs := pending[""]

	r.handler.Lock()
	var removed, added, changed []string
	for taDir := range r.handler.active {
		if _, ok := desired[taDir]; !ok {
			removed = append(removed, taDir)
		} else if allTAs {
			changed = append(changed, taDir)
		} else if _, ok := pending[taDir]; ok {
			changed = append(changed, taDir)
		}
	}
	for taDir := range desired {
		if _, ok := r.handler.active[taDir]; !ok {
			added = append(added, taDir)
		}
	}
	r.handler.Unlock()

	if len(removed) > 0 {
		r.handler.OnRemove(ctx, removed)
	}
	if len(added) > 0 {
		if err := r.handler.OnAdd(ctx, added); err != nil {
			logger.Error("splunk_inputs: failed to start receivers for new TAs", zap.Error(err))
		}
		for _, taDir := range added {
			_ = r.watcher.Add(filepath.Join(taDir, "default"))
			_ = r.watcher.Add(filepath.Join(taDir, "local"))
		}
	}
	if len(changed) > 0 {
		if err := r.handler.OnChange(ctx, changed); err != nil {
			logger.Error("splunk_inputs: failed to reload receivers for changed TAs", zap.Error(err))
		}
	}
}
