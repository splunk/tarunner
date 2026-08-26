// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/splunk/tarunner/internal/conf"
	"github.com/splunk/tarunner/internal/stanza"
	"github.com/splunk/tarunner/internal/tabuilder"
)

type (
	Input         = conf.Input
	Configuration = conf.Configuration
	Stanza        = conf.Stanza
	Params        = conf.Params
	Param         = conf.Param
	Prop          = conf.Prop
	Transform     = conf.Transform
	FieldAlias    = conf.FieldAlias
	PropType      = conf.PropType
)

// ReceiverRequest is passed to a sub-receiver factory for one inputs.conf
// stanza. Path is the parsed target from the stanza name. Empty-kind stanzas
// are dispatched to the "script" sub-receiver.
type ReceiverRequest struct {
	BaseDir    string
	Path       string
	Input      Input
	Transforms []Transform
	Props      []Prop
}

// SubReceiverFactory creates a logs receiver for one inputs.conf stanza kind.
//
// Scheme returns the stanza kind to match. Kinds are normalized to lower-case
// before matching. Returning "script" handles both script:// stanzas and empty-kind modular input
// stanzas.
type SubReceiverFactory interface {
	Scheme() string
	CreateLogs(context.Context, receiver.Settings, ReceiverRequest, consumer.Logs) (receiver.Logs, error)
}

// Option configures the splunk_inputs factory.
type Option func(*factoryOptions)

// WithSubReceiver registers a sub-receiver factory by Scheme. If another
// factory is already registered for the same scheme, it is replaced.
func WithSubReceiver(f SubReceiverFactory) Option {
	return func(o *factoryOptions) {
		if f == nil {
			return
		}
		o.subReceivers[strings.ToLower(f.Scheme())] = f
	}
}

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

	splunkHome, err := resolveSplunkHome(cfg)
	if err != nil {
		return nil, err
	}

	taDirs, err := discoverTAs(splunkHome)
	if err != nil {
		return nil, err
	}

	var allReceivers []receiver.Logs
	for _, taDir := range taDirs {
		dirs := tabuilder.ConfDirsWithSystem(splunkHome, taDir)
		inputs, err := tabuilder.ReadInputs(dirs)
		if err != nil {
			return nil, err
		}
		transforms, err := tabuilder.ReadTransforms(dirs)
		if err != nil {
			return nil, err
		}
		props, err := tabuilder.ReadProps(dirs)
		if err != nil {
			return nil, err
		}
		receivers, err := o.createReceivers(ctx, inputs, transforms, props, taDir, logs, settings)
		if err != nil {
			return nil, err
		}
		allReceivers = append(allReceivers, receivers...)
	}
	return packReceivers(allReceivers), nil
}

// discoverTAs returns all direct child directories under $SPLUNK_HOME/etc/apps
// whose name starts with "splunk_ta_" (case-insensitive).
func discoverTAs(splunkHome string) ([]string, error) {
	appsDir := filepath.Join(splunkHome, "etc", "apps")
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("splunk_inputs: failed to scan %s: %w", appsDir, err)
	}
	var taDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(strings.ToLower(entry.Name()), "splunk_ta_") {
			taDirs = append(taDirs, filepath.Join(appsDir, entry.Name()))
		}
	}
	return taDirs, nil
}

func (o factoryOptions) createReceivers(ctx context.Context, inputs []Input, transforms []Transform, props []Prop, baseDir string, next consumer.Logs, settings receiver.Settings) ([]receiver.Logs, error) {
	var receivers []receiver.Logs
	for _, input := range inputs {
		name := input.Configuration.Stanza.Name
		if input.Configuration.Stanza.IsDisabled() {
			settings.Logger.Info("splunk_inputs: skipping disabled stanza", zap.String("stanza", name))
			continue
		}
		l, err := o.createReceiver(ctx, baseDir, next, input, transforms, props, settings)
		if err != nil {
			return nil, fmt.Errorf("failed to create receiver %q: %w", name, err)
		}
		if l == nil {
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
	if parsed.Target == "" {
		// stanza has no target (e.g. [script], [tcp]) — it is a default-settings
		// stanza, not an actual input to run.
		return nil, nil
	}
	scheme := parsed.Kind
	if scheme == "" {
		scheme = "script"
	}
	if f, ok := o.subReceivers[scheme]; ok {
		return f.CreateLogs(ctx, settings, ReceiverRequest{
			BaseDir:    baseDir,
			Path:       parsed.Target,
			Input:      input,
			Transforms: transforms,
			Props:      props,
		}, next)
	}
	l, err := tabuilder.CreateReceiver(ctx, baseDir, next, input, transforms, props, settings.TelemetrySettings)
	if l == nil && err == nil {
		return nil, fmt.Errorf("unsupported scheme %q", scheme)
	}
	return l, err
}
