// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package process

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver"
	"go.opentelemetry.io/collector/receiver"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"github.com/splunk/tarunner/internal/conf"
)

type Runner struct {
	doneChan       chan struct{}
	logger         *zap.Logger
	next           consumer.Logs
	commands       []*exec.Cmd
	filelog        receiver.Logs
	monitoredPaths []string
}

func NewRunner(next consumer.Logs, logger *zap.Logger) *Runner {
	return &Runner{
		doneChan: make(chan struct{}),
		logger:   logger,
		next:     next,
	}
}

func (r *Runner) Run(baseDir string) (bool, error) {
	fileToRead := filepath.Join(baseDir, "local", "inputs.conf")
	if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
		fileToRead = filepath.Join(baseDir, "default", "inputs.conf")
		if _, err := os.Stat(fileToRead); errors.Is(err, os.ErrNotExist) {
			return false, err
		}
	}
	b, err := os.ReadFile(fileToRead)
	if err != nil {
		return false, err
	}
	inputs, err := conf.ReadInput(b)
	if err != nil {
		return false, err
	}
	result := false
	for _, input := range inputs {
		schedulingResult, err := r.scheduleInput(baseDir, input)
		result = result || schedulingResult
		if err != nil {
			return false, err
		}
	}
	if len(r.monitoredPaths) > 0 {
		f := filelogreceiver.NewFactory()
		cfg := f.CreateDefaultConfig().(*filelogreceiver.FileLogConfig)
		cfg.InputConfig.Include = r.monitoredPaths
		r.filelog, err = f.CreateLogs(context.Background(), receiver.Settings{
			TelemetrySettings: component.TelemetrySettings{
				Logger:         r.logger,
				TracerProvider: noop.NewTracerProvider(),
				MeterProvider:  metricnoop.NewMeterProvider(),
				Resource:       pcommon.NewResource(),
			},
		}, cfg, r.next)
		if err != nil {
			return false, err
		}
	}
	return result, nil
}

func (r *Runner) scheduleInput(baseDir string, input conf.Input) (bool, error) {
	parsed, err := url.Parse(input.Configuration.Stanza.Name)
	if err != nil {
		return false, err
	}
	switch parsed.Scheme {
	case "script":
		return r.scheduleScriptedInput(baseDir, input)
	case "":
		return r.scheduleScriptedInput(baseDir, input)
	case "monitor":
		return r.addPath(baseDir, input)
	default:
		return false, fmt.Errorf("unknown scheme %q", parsed.Scheme)
	}
}

func (r *Runner) scheduleScriptedInput(baseDir string, input conf.Input) (bool, error) {
	intervalS := 3600
	for _, p := range input.Configuration.Stanza.Params {
		if p.Name == "interval" {
			var err error
			intervalS, err = strconv.Atoi(p.Value)
			if err != nil {
				return false, err
			}
		}
		if p.Name == "disabled" && p.Value == "1" {
			return false, nil
		}
	}
	if intervalS == -1 {
		return false, nil
	}
	if intervalS == 0 {
		go func() {
			for {
				select {
				case <-r.doneChan:
					return
				default:
					r.execute(baseDir, input)
				}
			}
		}()
	} else {
		interval := time.Duration(intervalS) * time.Second
		go func() {
			r.execute(baseDir, input)

			for {
				select {
				case <-time.After(interval):
					r.execute(baseDir, input)
				case <-r.doneChan:
					return
				}
			}
		}()
	}
	return true, nil
}

func (r *Runner) execute(baseDir string, input conf.Input) {
	if err := r._execute(baseDir, input); err != nil {
		r.logger.Error("Error executing input", zap.String("input", input.Configuration.Stanza.Name), zap.Error(err))
	}
}

func (r *Runner) _execute(baseDir string, input conf.Input) error {
	command, err := determineCommandName(baseDir, input)
	if err != nil {
		return err
	}
	cmd := exec.Command(command)
	var stdin io.WriteCloser
	var stdout io.ReadCloser
	var inputXML []byte
	if stdin, err = cmd.StdinPipe(); err != nil {
		return err
	}
	if stdout, err = cmd.StdoutPipe(); err != nil {
		return err
	}
	stdoutReader := bufio.NewReader(stdout)
	if inputXML, err = input.ToXML(); err != nil {
		return err
	}
	if _, err = stdin.Write(inputXML); err != nil {
		return err
	}
	if err = stdin.Close(); err != nil {
		return err
	}

	index := "main"
	if indexParam := input.Configuration.Stanza.Params.Get("index"); indexParam != nil {
		index = indexParam.Value
	}

	sourcetype := ""
	if sourceTypeParam := input.Configuration.Stanza.Params.Get("sourcetype"); sourceTypeParam != nil {
		sourcetype = sourceTypeParam.Value
	}

	go func() {
		for {
			select {
			case <-r.doneChan:
				return
			default:
				b, err := stdoutReader.ReadBytes('\n')
				if err != nil {
					return
				} else {
					logs := plog.NewLogs()
					rl := logs.ResourceLogs().AppendEmpty()
					rl.Resource().Attributes().PutStr("com.splunk.index", index)
					rl.Resource().Attributes().PutStr("com.splunk.source", input.Configuration.Stanza.Name)
					rl.Resource().Attributes().PutStr("com.splunk.sourcetype", sourcetype)
					rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr(string(b))
					if err = r.next.ConsumeLogs(context.Background(), logs); err != nil {
						r.logger.Error("Error consuming logs", zap.Error(err))
					}
				}
			}
		}
	}()

	if err = cmd.Start(); err != nil {
		return err
	}
	r.commands = append(r.commands, cmd)

	return cmd.Wait()
}

func determineCommandName(baseDir string, input conf.Input) (string, error) {
	parsed, err := url.Parse(input.Configuration.Stanza.Name)
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "script":
		return filepath.Join(baseDir, parsed.Path), nil
	case "":
		return filepath.Join(baseDir, "bin", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH), input.Configuration.Stanza.Name), nil
	default:
		return "", fmt.Errorf("unknown scheme %q", parsed.Scheme)
	}
}

func (r *Runner) Shutdown() {
	for _, cmd := range r.commands {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}
	if r.filelog != nil {
		_ = r.filelog.Shutdown(context.Background())
	}
	close(r.doneChan)
}

func (r *Runner) Done() <-chan struct{} {
	return r.doneChan
}

func (r *Runner) addPath(baseDir string, input conf.Input) (bool, error) {
	parsed, err := url.Parse(input.Configuration.Stanza.Name)
	if err != nil {
		return false, err
	}
	if filepath.IsAbs(parsed.Path) {
		r.monitoredPaths = append(r.monitoredPaths, parsed.Path)
	} else {
		r.monitoredPaths = append(r.monitoredPaths, filepath.Join(baseDir, parsed.Path))
	}
	return true, nil
}
