// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

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

	receivers, err := o.createReceivers(ctx, inputs, transforms, props, baseDir, logs, settings)
	if err != nil {
		return nil, err
	}
	return packReceivers(receivers), nil
}

func (o factoryOptions) createReceivers(ctx context.Context, inputs []Input, transforms []Transform, props []Prop, baseDir string, next consumer.Logs, settings receiver.Settings) ([]receiver.Logs, error) {
	var receivers []receiver.Logs
	for _, input := range inputs {
		disabled := input.Configuration.Stanza.Params.Get("disabled")
		if disabled != nil && disabled.Value == "1" {
			continue
		}
		l, err := o.createReceiver(ctx, baseDir, next, input, transforms, props, settings)
		if err != nil {
			return nil, fmt.Errorf("failed to create receiver %q: %w", input.Configuration.Stanza.Name, err)
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
	if f, ok := o.subReceivers[scheme]; ok {
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
