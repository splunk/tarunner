// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package scriptedinput

import (
	"runtime"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/splunk/tarunner/internal/conf"
)

func Test_ScriptedInput_PermanentError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows because scripts use bash")
	}

	core, logs := observer.New(zapcore.ErrorLevel)
	settings := componenttest.NewNopTelemetrySettings()
	settings.Logger = zap.New(core)

	c := NewConfig()
	c.BaseDir = "testdata"
	c.Input = conf.Input{
		Configuration: conf.Configuration{
			Stanza: conf.Stanza{
				Name:   "script://./bin/nonexistent.sh",
				Params: []conf.Param{{Name: "interval", Value: "1"}},
			},
		},
	}
	o, err := c.Build(settings)
	require.NoError(t, err)
	fo := testutil.NewFakeOutput(t)
	require.NoError(t, fo.Start(nil))
	t.Cleanup(func() { require.NoError(t, fo.Stop()) })
	o.SetOutputIDs([]string{fo.ID()})
	require.NoError(t, o.SetOutputs([]operator.Operator{fo}))
	require.NoError(t, o.Start(nil))

	// wait long enough that a non-permanent error would have retried multiple times
	time.Sleep(300 * time.Millisecond)
	require.NoError(t, o.Stop())

	// permanent error should have been logged exactly once, not retried
	assert.Equal(t, 1, logs.Len(), "expected exactly one error log, got %d", logs.Len())
	assert.Len(t, fo.Received, 0)
}

func Test_ScriptedInput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows because scripts use bash")
	}

	tests := []struct {
		name      string
		interval  string
		expectMsg bool
	}{
		{
			"always",
			"0",
			true,
		},
		{
			"polling",
			"1",
			true,
		},
		{
			"disabled",
			"-1",
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := NewConfig()
			c.BaseDir = "testdata"
			c.Input = conf.Input{
				Configuration: conf.Configuration{
					Stanza: conf.Stanza{
						Name: "script://./bin/foo.sh",
						Params: []conf.Param{
							{Name: "interval", Value: test.interval},
						},
					},
				},
			}
			settings := componenttest.NewNopTelemetrySettings()
			settings.Logger, _ = zap.NewDevelopment()
			o, err := c.Build(settings)
			assert.NoError(t, err)
			require.NotNil(t, o)
			fo := testutil.NewFakeOutput(t)
			require.NoError(t, fo.Start(nil))
			t.Cleanup(func() {
				require.NoError(t, fo.Stop())
			})
			o.SetOutputIDs([]string{fo.ID()})
			err = o.SetOutputs([]operator.Operator{
				fo,
			})
			require.NoError(t, err)
			err = o.Start(nil)
			require.NoError(t, err)
			if test.expectMsg {
				select {
				case msg := <-fo.Received:
					require.NotNil(t, msg)
					require.Equal(t, "foo\n", msg.Body)
				case <-time.After(5 * time.Second):
					require.Fail(t, "timed out waiting for message")
				}
			} else {
				time.Sleep(time.Millisecond * 100)
				require.Len(t, fo.Received, 0)
			}
			require.NoError(t, o.Stop())
		})
	}
}
