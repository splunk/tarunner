// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestRunTA(t *testing.T) {
	logsSink := &consumertest.LogsSink{}
	cfg := otlpreceiver.NewFactory().CreateDefaultConfig().(*otlpreceiver.Config)
	http := cfg.HTTP.GetOrInsertDefault()
	http.ServerConfig.NetAddr.Endpoint = "localhost:1337"

	rcvr, err := otlpreceiver.NewFactory().CreateLogs(context.Background(), receivertest.NewNopSettings(otlpreceiver.NewFactory().Type()), cfg, logsSink)
	require.NoError(t, err)
	err = rcvr.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	defer func() {
		_ = rcvr.Shutdown(context.Background())
	}()
	cancel, err := Run(filepath.Join("testdata", "ta"), "http://localhost:1337")
	require.NoError(t, err)
	defer cancel()

	require.Eventually(t, func() bool {
		return logsSink.LogRecordCount() == 20
	}, 2*time.Second, 10*time.Millisecond)
}

func TestRunPeriodic(t *testing.T) {
	logsSink := &consumertest.LogsSink{}
	cfg := otlpreceiver.NewFactory().CreateDefaultConfig().(*otlpreceiver.Config)
	cfg.HTTP.GetOrInsertDefault().ServerConfig.NetAddr.Endpoint = "localhost:1337"
	rcvr, err := otlpreceiver.NewFactory().CreateLogs(context.Background(), receivertest.NewNopSettings(otlpreceiver.NewFactory().Type()), cfg, logsSink)
	require.NoError(t, err)
	err = rcvr.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	defer func() {
		_ = rcvr.Shutdown(context.Background())
	}()
	cancel, err := Run(filepath.Join("testdata", "periodic"), "http://localhost:1337")
	require.NoError(t, err)
	defer cancel()

	require.Eventually(t, func() bool {
		return logsSink.LogRecordCount() == 10
	}, 1*time.Second, 10*time.Millisecond)

	result := make([]string, 0, 10)
	for _, log := range logsSink.AllLogs() {
		for _, rl := range log.ResourceLogs().All() {
			for _, sl := range rl.ScopeLogs().All() {
				for _, lr := range sl.LogRecords().All() {
					result = append(result, lr.Body().Str())
				}
			}
		}
	}

	require.Equal(t, []string{"foo1\n", "foo2\n", "foo3\n", "foo4\n", "foo5\n", "foo6\n", "foo7\n", "foo8\n", "foo9\n", "foo10\n"}, result)

	// reset and get a second run:
	result = make([]string, 0, 10)
	for _, log := range logsSink.AllLogs() {
		for _, rl := range log.ResourceLogs().All() {
			for _, sl := range rl.ScopeLogs().All() {
				for _, lr := range sl.LogRecords().All() {
					result = append(result, lr.Body().Str())
				}
			}
		}
	}

	require.Equal(t, []string{"foo1\n", "foo2\n", "foo3\n", "foo4\n", "foo5\n", "foo6\n", "foo7\n", "foo8\n", "foo9\n", "foo10\n"}, result)
}

func TestRunDisabled(t *testing.T) {
	logsSink := &consumertest.LogsSink{}
	cfg := otlpreceiver.NewFactory().CreateDefaultConfig().(*otlpreceiver.Config)
	cfg.HTTP.GetOrInsertDefault().ServerConfig.NetAddr.Endpoint = "localhost:1337"
	rcvr, err := otlpreceiver.NewFactory().CreateLogs(context.Background(), receivertest.NewNopSettings(otlpreceiver.NewFactory().Type()), cfg, logsSink)
	require.NoError(t, err)
	err = rcvr.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	defer func() {
		_ = rcvr.Shutdown(context.Background())
	}()
	cancel, err := Run(filepath.Join("testdata", "disabled"), "http://localhost:1337")
	require.NoError(t, err)
	require.Nil(t, cancel)
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0, logsSink.LogRecordCount())
}

func TestRunDisabledInterval(t *testing.T) {
	logsSink := &consumertest.LogsSink{}
	cfg := otlpreceiver.NewFactory().CreateDefaultConfig().(*otlpreceiver.Config)
	cfg.HTTP.GetOrInsertDefault().ServerConfig.NetAddr.Endpoint = "localhost:1337"
	rcvr, err := otlpreceiver.NewFactory().CreateLogs(context.Background(), receivertest.NewNopSettings(otlpreceiver.NewFactory().Type()), cfg, logsSink)
	require.NoError(t, err)
	err = rcvr.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	defer func() {
		_ = rcvr.Shutdown(context.Background())
	}()
	cancel, err := Run(filepath.Join("testdata", "disabled_interval"), "http://localhost:1337")
	require.NoError(t, err)
	defer cancel()

	assert.Equal(t, logsSink.LogRecordCount(), 0)
}

func TestRunScriptedInputs(t *testing.T) {
	logsSink := &consumertest.LogsSink{}
	cfg := otlpreceiver.NewFactory().CreateDefaultConfig().(*otlpreceiver.Config)
	cfg.HTTP.GetOrInsertDefault().ServerConfig.NetAddr.Endpoint = "localhost:1337"
	rcvr, err := otlpreceiver.NewFactory().CreateLogs(context.Background(), receivertest.NewNopSettings(otlpreceiver.NewFactory().Type()), cfg, logsSink)
	require.NoError(t, err)
	err = rcvr.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	defer func() {
		_ = rcvr.Shutdown(context.Background())
	}()
	cancel, err := Run(filepath.Join("testdata", "script"), "http://localhost:1337")
	require.NoError(t, err)
	defer cancel()

	require.Eventually(t, func() bool {
		return logsSink.LogRecordCount() == 20
	}, 2*time.Second, 10*time.Millisecond)
}

func TestReadTransforms(t *testing.T) {
	rootDir := filepath.Join("testdata", "transforms")
	tests := []struct {
		name          string
		path          string
		expectedName  string
		expectedRegex string
	}{
		{
			name:          "default",
			path:          filepath.Join(rootDir, "default"),
			expectedName:  "example_default",
			expectedRegex: "default",
		},
		{
			name:          "local",
			path:          filepath.Join(rootDir, "local"),
			expectedName:  "example_local",
			expectedRegex: "local",
		},
		{
			name:          "both",
			path:          filepath.Join(rootDir, "both"),
			expectedName:  "example_local2",
			expectedRegex: "local",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transforms, err := readTransforms(test.path)
			require.NoError(t, err)
			require.Len(t, transforms, 1)
			require.Equal(t, test.expectedName, transforms[0].Name)
			require.Equal(t, test.expectedRegex, transforms[0].Regex)
		})
	}
}
