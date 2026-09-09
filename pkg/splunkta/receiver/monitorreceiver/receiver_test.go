// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package monitorreceiver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/pipeline"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/testutil"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"github.com/splunk/tarunner/pkg/splunkta/conf"
)

// TestMonitorDirectoryWithSplunkRegexWhitelist tests the exact Splunk_TA_nix default stanza
// merged with a local overlay that only sets disabled=0 and index:
//
//	[monitor:///var/log]                          # default TA layer
//	whitelist=(\.log|log$|messages|secure|auth|mesg$|cron$|acpid$|\.out)
//	blacklist=(lastlog|anaconda\.syslog)
//
// The whitelist/blacklist values are PCRE regexes, not globs. The receiver detects
// this and falls back to dir/* so that all files are ingested (the TA overlay is
// expected to narrow scope via a glob whitelist if needed, but must not break the
// common case where no overlay whitelist is set).
func TestMonitorDirectoryWithSplunkRegexWhitelist(t *testing.T) {
	tempDir := t.TempDir()

	// Exact values from Splunk_TA_nix default/inputs.conf
	const whitelist = `(\.log|log$|messages|secure|auth|mesg$|cron$|acpid$|\.out)`
	const blacklist = `(lastlog|anaconda\.syslog)`

	cfg := Config{
		Input: conf.Input{
			Configuration: conf.Configuration{
				Stanza: conf.Stanza{
					Name: fmt.Sprintf("monitor://%s", tempDir),
					Params: conf.Params{
						conf.Param{Name: "whitelist", Value: whitelist},
						conf.Param{Name: "blacklist", Value: blacklist},
						conf.Param{Name: "index", Value: "otel_nix"},
					},
				},
			},
		},
	}
	logger, _ := zap.NewDevelopment()
	c := monitor{logger: logger}.InputConfig(cfg)
	o, err := c.Build(component.TelemetrySettings{
		Logger:         logger,
		TracerProvider: nooptrace.NewTracerProvider(),
		MeterProvider:  noopmetric.NewMeterProvider(),
		Resource:       pcommon.NewResource(),
	})
	require.NoError(t, err)
	output := testutil.NewFakeOutput(t)
	o.SetOutputIDs([]string{"fake"})
	require.NoError(t, o.SetOutputs([]operator.Operator{output}))
	require.NoError(t, o.Start(nil))
	defer func() { require.NoError(t, o.Stop()) }()

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "syslog.log"), []byte("line1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "auth"), []byte("line2\n"), 0o644))

	// The regex whitelist is not a valid glob; the receiver falls back to dir/*
	// so both files must be ingested.
	received := map[string]bool{}
	deadline := time.After(3 * time.Second)
	for len(received) < 2 {
		select {
		case got := <-output.Received:
			name, _ := got.Attributes["log.file.name"].(string)
			received[name] = true
		case <-deadline:
			t.Fatalf("timeout waiting for files; got: %v", received)
		}
	}
	require.True(t, received["syslog.log"], "expected syslog.log to be ingested")
	require.True(t, received["auth"], "expected auth to be ingested")
}

// TestMonitorDirectoryEmptyWhitelist mirrors the real-life Splunk_TA_nix stanza after the
// overlay sets whitelist= (empty) to bypass the Splunk regex value that is not a valid glob:
//
//	[monitor:///var/log]
//	whitelist=
//	blacklist=
//
// An empty whitelist on a plain directory path must expand to dir/* so that filelog
// can actually match files. Without this expansion the operator receives a bare directory
// path which never matches any file.
func TestMonitorDirectoryEmptyWhitelist(t *testing.T) {
	tempDir := t.TempDir()

	cfg := Config{
		Input: conf.Input{
			Configuration: conf.Configuration{
				Stanza: conf.Stanza{
					Name: fmt.Sprintf("monitor://%s", tempDir),
					Params: conf.Params{
						conf.Param{Name: "whitelist", Value: ""},
						conf.Param{Name: "blacklist", Value: ""},
						conf.Param{Name: "index", Value: "otel_nix"},
					},
				},
			},
		},
	}
	logger, _ := zap.NewDevelopment()
	c := monitor{logger: logger}.InputConfig(cfg)
	o, err := c.Build(component.TelemetrySettings{
		Logger:         logger,
		TracerProvider: nooptrace.NewTracerProvider(),
		MeterProvider:  noopmetric.NewMeterProvider(),
		Resource:       pcommon.NewResource(),
	})
	require.NoError(t, err)
	output := testutil.NewFakeOutput(t)
	o.SetOutputIDs([]string{"fake"})
	require.NoError(t, o.SetOutputs([]operator.Operator{output}))
	require.NoError(t, o.Start(nil))
	defer func() { require.NoError(t, o.Stop()) }()

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "syslog"), []byte("hello log\n"), 0o644))
	received := <-output.Received
	require.Equal(t, "hello log\n", received.Body)
	// InputConfig sets the raw "index" attribute; renameMetadata (wired by the adapter)
	// moves it to "com.splunk.index" in the full pipeline.
	require.Equal(t, "otel_nix", received.Attributes["index"])
}

func TestReadFile(t *testing.T) {
	tempDir := t.TempDir()

	cfg := Config{
		Input: conf.Input{
			Configuration: conf.Configuration{
				Stanza: conf.Stanza{
					Name: fmt.Sprintf("monitor://%s%c%s", tempDir, filepath.Separator, "foo.txt"),
					App:  "",
					Params: conf.Params{
						conf.Param{
							Name:  "host",
							Value: "myhost",
						},
					},
				},
			},
		},
	}
	logger, _ := zap.NewDevelopment()
	c := monitor{logger: logger}.InputConfig(cfg)
	o, err := c.Build(component.TelemetrySettings{
		Logger:         logger,
		TracerProvider: nooptrace.NewTracerProvider(),
		MeterProvider:  noopmetric.NewMeterProvider(),
		Resource:       pcommon.NewResource(),
	})
	require.NoError(t, err)
	output := testutil.NewFakeOutput(t)
	o.SetOutputIDs([]string{"fake"})
	require.NoError(t, o.SetOutputs([]operator.Operator{
		output,
	}))
	err = o.Start(nil)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, o.Stop())
	}()

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "foo.txt"), []byte("foo\n"), 0o644))
	received := <-output.Received
	require.Equal(t, "foo\n", received.Body)
}

func TestRenameMetadata(t *testing.T) {
	ops := renameMetadata()
	output := testutil.NewFakeOutput(t)
	pipe, err := pipeline.Config{
		Operators:     ops,
		DefaultOutput: output,
	}.Build(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	require.NoError(t, pipe.Start(nil))
	defer func() {
		require.NoError(t, pipe.Stop())
	}()
	require.NoError(t, pipe.Operators()[0].Process(context.Background(), &entry.Entry{
		Attributes: map[string]any{
			"source":     "src",
			"sourcetype": "srctype",
			"host":       "foo",
		},
	}))

	result := <-output.Received
	require.Equal(t, "src", result.Attributes["com.splunk.source"])
	require.Equal(t, "srctype", result.Attributes["com.splunk.sourcetype"])
	require.Equal(t, "foo", result.Attributes["host.name"])
}
