// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkoutputsexporter_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/splunk/tarunner/pkg/splunkoutputsexporter"
)

func TestWithSubExporterOverridesBuiltInHTTPOut(t *testing.T) {
	baseDir := writeTA(t, "[httpout]\nuri = https://example.com/services/collector/event\nhttpEventCollectorToken = token\n")
	fake := &fakeSubExporterFactory{scheme: "httpout"}

	factory := splunkoutputsexporter.NewFactory(splunkoutputsexporter.WithSubExporter(fake))
	exp, err := factory.CreateLogs(context.Background(), newExporterSettings(), splunkoutputsexporter.Config{BaseDir: baseDir})
	require.NoError(t, err)
	require.NotNil(t, exp)
	require.Len(t, fake.requests, 1)
	require.Equal(t, baseDir, fake.requests[0].BaseDir)
	require.Equal(t, "token", fake.requests[0].Output.Token)
	require.Equal(t, "https://example.com/services/collector/event", fake.requests[0].Output.URI)
}

func TestWithSubExporterOnlyReadsHTTPOut(t *testing.T) {
	baseDir := writeTA(t, "[httpout]\nuri = https://example.com/services/collector/event\nhttpEventCollectorToken = token\n\n[tcpout:primary]\nserver = splunk:9997\n")
	httpout := &fakeSubExporterFactory{scheme: "httpout"}
	tcpout := &fakeSubExporterFactory{scheme: "tcpout"}

	factory := splunkoutputsexporter.NewFactory(
		splunkoutputsexporter.WithSubExporter(httpout),
		splunkoutputsexporter.WithSubExporter(tcpout),
	)
	exp, err := factory.CreateLogs(context.Background(), newExporterSettings(), splunkoutputsexporter.Config{BaseDir: baseDir})
	require.NoError(t, err)
	require.NotNil(t, exp)
	require.Len(t, httpout.requests, 1)
	require.Empty(t, tcpout.requests)
}

func TestWithSubExporterRequiresHTTPOut(t *testing.T) {
	baseDir := writeTA(t, "[tcpout:primary]\nserver = splunk:9997\n")
	fake := &fakeSubExporterFactory{scheme: "tcpout"}

	factory := splunkoutputsexporter.NewFactory(splunkoutputsexporter.WithSubExporter(fake))
	exp, err := factory.CreateLogs(context.Background(), newExporterSettings(), splunkoutputsexporter.Config{BaseDir: baseDir})
	require.ErrorContains(t, err, `no [httpout] stanza found in outputs.conf`)
	require.Nil(t, exp)
	require.Empty(t, fake.requests)
}

type fakeSubExporterFactory struct {
	scheme   string
	requests []splunkoutputsexporter.ExporterRequest
}

func (f *fakeSubExporterFactory) Scheme() string {
	return f.scheme
}

func (f *fakeSubExporterFactory) CreateLogs(_ context.Context, _ exporter.Settings, request splunkoutputsexporter.ExporterRequest) (exporter.Logs, error) {
	f.requests = append(f.requests, request)
	return fakeLogsExporter{}, nil
}

type fakeLogsExporter struct{}

func (fakeLogsExporter) Start(context.Context, component.Host) error {
	return nil
}

func (fakeLogsExporter) Shutdown(context.Context) error {
	return nil
}

func (fakeLogsExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func (fakeLogsExporter) ConsumeLogs(context.Context, plog.Logs) error {
	return nil
}

func newExporterSettings() exporter.Settings {
	return exporter.Settings{
		ID: component.MustNewID("splunk_outputs"),
	}
}

func writeTA(t *testing.T, outputsConf string) string {
	t.Helper()
	baseDir := t.TempDir()
	defaultDir := filepath.Join(baseDir, "default")
	require.NoError(t, os.Mkdir(defaultDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(defaultDir, "outputs.conf"), []byte(outputsConf), 0o600))
	return baseDir
}