package tareceiver

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/splunk/tarunner/internal/conf"
	"github.com/splunk/tarunner/internal/receiver/monitorreceiver"
	"github.com/splunk/tarunner/internal/receiver/scriptreceiver"
	"github.com/splunk/tarunner/internal/receiver/tcpreceiver"
	"github.com/splunk/tarunner/internal/receiver/udpreceiver"
	"github.com/splunk/tarunner/internal/receiver/wineventlogreceiver"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
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

func createLogsFunc(_ context.Context, settings receiver.Settings, config component.Config, logs consumer.Logs) (receiver.Logs, error) {
	cfg := config.(Config)
	baseDir := cfg.BaseDir
	inputs, err := readInputs(baseDir)
	if err != nil {
		return nil, err
	}
	transforms, err := readTransforms(baseDir)
	if err != nil {
		return nil, err
	}
	props, err := readProps(baseDir)
	if err != nil {
		return nil, err
	}

	receivers, err := createReceivers(inputs, transforms, props, baseDir, logs, settings.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return packReceivers(receivers), nil
}

func createReceivers(inputs []conf.Input, transforms []conf.Transform, props []conf.Prop, baseDir string, next consumer.Logs, telemetrySettings component.TelemetrySettings) ([]receiver.Logs, error) {
	var receivers []receiver.Logs
	for _, input := range inputs {
		disabled := input.Configuration.Stanza.Params.Get("disabled")
		if disabled != nil && disabled.Value == "1" {
			continue
		}
		l, err := createReceiver(baseDir, next, input, transforms, props, telemetrySettings)
		if err != nil {
			return nil, fmt.Errorf("failed to create receiver %q: %w", input.Configuration.Stanza.Name, err)
		}
		receivers = append(receivers, l)
	}
	return receivers, nil
}

func readInputs(baseDir string) ([]conf.Input, error) {
	fileToRead := filepath.Join(baseDir, "local", "inputs.conf")
	if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
		fileToRead = filepath.Join(baseDir, "default", "inputs.conf")
		if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	b, err := os.ReadFile(fileToRead)
	if err != nil {
		return nil, err
	}
	return conf.ReadInput(b)
}

func readTransforms(baseDir string) ([]conf.Transform, error) {
	fileToRead := filepath.Join(baseDir, "local", "transforms.conf")
	if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
		fileToRead = filepath.Join(baseDir, "default", "transforms.conf")
		if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
	}
	b, err := os.ReadFile(fileToRead)
	if err != nil {
		return nil, err
	}
	return conf.ReadTransforms(b)
}

func readProps(baseDir string) ([]conf.Prop, error) {
	fileToRead := filepath.Join(baseDir, "local", "props.conf")
	if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
		fileToRead = filepath.Join(baseDir, "default", "props.conf")
		if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
	}
	b, err := os.ReadFile(fileToRead)
	if err != nil {
		return nil, err
	}
	return conf.ReadProps(b)
}

func createReceiver(baseDir string, next consumer.Logs, input conf.Input, transforms []conf.Transform, props []conf.Prop, telemetrySettings component.TelemetrySettings) (receiver.Logs, error) {
	parsed, err := url.Parse(input.Configuration.Stanza.Name)
	if err != nil {
		return nil, err
	}
	switch parsed.Scheme {
	case "script", "":
		f := scriptreceiver.NewFactory()
		l, err := f.CreateLogs(context.Background(), receiver.Settings{
			ID:                component.MustNewIDWithName(f.Type().String(), parsed.Path),
			TelemetrySettings: telemetrySettings,
		}, &scriptreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		},
			next)
		return l, err
	case "monitor":
		f := monitorreceiver.NewFactory()
		l, err := f.CreateLogs(context.Background(), receiver.Settings{
			ID:                component.MustNewIDWithName(f.Type().String(), parsed.Path),
			TelemetrySettings: telemetrySettings,
		}, monitorreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		},
			next)
		return l, err
	case "WinEventLog":
		f := wineventlogreceiver.NewFactory()
		l, err := f.CreateLogs(context.Background(), receiver.Settings{
			ID:                component.MustNewIDWithName(f.Type().String(), parsed.Path),
			TelemetrySettings: telemetrySettings,
		}, wineventlogreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		},
			next)
		return l, err
	case "tcp":
		f := tcpreceiver.NewFactory()
		l, err := f.CreateLogs(context.Background(), receiver.Settings{
			ID:                component.MustNewIDWithName(f.Type().String(), parsed.Path),
			TelemetrySettings: telemetrySettings,
		}, tcpreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		},
			next)
		return l, err
	case "udp":
		f := udpreceiver.NewFactory()
		l, err := f.CreateLogs(context.Background(), receiver.Settings{
			ID:                component.MustNewIDWithName(f.Type().String(), parsed.Path),
			TelemetrySettings: telemetrySettings,
		}, udpreceiver.Config{
			Input:      input,
			BaseDir:    baseDir,
			Transforms: transforms,
			Props:      props,
		},
			next)
		return l, err
	default:
		return nil, fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
}
