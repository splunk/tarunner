// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package scriptedinput

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/splunk/tarunner/internal/script"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"go.uber.org/zap"

	"github.com/splunk/tarunner/internal/conf"
)

type ScriptedInput struct {
	logger   *zap.Logger
	doneChan chan struct{}
	command  *exec.Cmd
	cfg      Config
	helper.InputOperator
}

func (si *ScriptedInput) Start(_ operator.Persister) error {
	if _, err := si.scheduleInput(si.cfg.BaseDir, si.cfg.Input); err != nil {
		return err
	}
	return nil
}

func (si *ScriptedInput) Stop() error {
	if si.command != nil {
		_ = si.command.Process.Signal(syscall.SIGTERM)
	}
	close(si.doneChan)

	return nil
}

func (si *ScriptedInput) scheduleInput(baseDir string, input conf.Input) (bool, error) {
	parsed, err := url.Parse(input.Configuration.Stanza.Name)
	if err != nil {
		return false, err
	}
	switch parsed.Scheme {
	case "script":
		return si.scheduleScriptedInput(baseDir, input)
	case "":
		return si.scheduleScriptedInput(baseDir, input)
	default:
		return false, fmt.Errorf("unknown scheme %q", parsed.Scheme)
	}
}

func (si *ScriptedInput) scheduleScriptedInput(baseDir string, input conf.Input) (bool, error) {
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
				case <-si.doneChan:
					return
				default:
					si.execute(baseDir, input)
				}
			}
		}()
	} else {
		interval := time.Duration(intervalS) * time.Second
		go func() {
			si.execute(baseDir, input)

			for {
				select {
				case <-time.After(interval):
					si.execute(baseDir, input)
				case <-si.doneChan:
					return
				}
			}
		}()
	}
	return true, nil
}

func (si *ScriptedInput) execute(baseDir string, input conf.Input) {
	if err := si._execute(baseDir, input); err != nil {
		si.logger.Error("Error executing input", zap.String("input", input.Configuration.Stanza.Name), zap.Error(err))
	}
}

func (si *ScriptedInput) _execute(baseDir string, input conf.Input) error {
	command, err := script.DetermineCommandName(baseDir, input)
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

	go func() {
		for {
			select {
			case <-si.doneChan:
				return
			default:
				b, err := io.ReadAll(stdoutReader)
				if err != nil {
					si.logger.Error("Error reading log data", zap.Error(err))
					return
				} else {
					e := entry.New()
					e.Body = string(b)
					if err := si.Attributer.Attribute(e); err != nil {
						si.logger.Error("Error setting attributes", zap.Error(err))
					}

					if err = si.Write(context.Background(), e); err != nil {
						si.logger.Error("Error consuming logs", zap.Error(err))
					}
				}
			}
		}
	}()

	if err = cmd.Start(); err != nil {
		return err
	}
	si.command = cmd

	return cmd.Wait()
}
