// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunktaobserver

import (
	"context"
	"os"
	"path/filepath"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer/endpointswatcher"
)

// TADirectoryType identifies an endpoint as a Splunk TA directory.
const TADirectoryType observer.EndpointType = "splunk.ta_directory"

// TADirectory holds details about a discovered TA directory endpoint.
type TADirectory struct {
	// Name is the TA folder name, e.g. "Splunk_TA_nix".
	Name string
	// Path is the absolute path to the TA directory.
	Path string
}

func (t *TADirectory) Type() observer.EndpointType { return TADirectoryType }
func (t *TADirectory) Env() observer.EndpointEnv {
	return observer.EndpointEnv{
		"name": t.Name,
		"path": t.Path,
	}
}

type taObserver struct {
	*endpointswatcher.EndpointsWatcher
}

var _ extension.Extension = (*taObserver)(nil)

func newObserver(params extension.Settings, config *Config) (extension.Extension, error) {
	return &taObserver{
		EndpointsWatcher: endpointswatcher.New(
			&taLister{path: config.Path, logger: params.Logger},
			config.RefreshInterval,
			params.Logger,
		),
	}, nil
}

func (*taObserver) Start(context.Context, component.Host) error { return nil }

func (o *taObserver) Shutdown(context.Context) error {
	o.StopListAndWatch()
	return nil
}

type taLister struct {
	path   string
	logger *zap.Logger
}

func (l *taLister) ListEndpoints() []observer.Endpoint {
	entries, err := os.ReadDir(l.path)
	if err != nil {
		l.logger.Error("failed to read TA directory", zap.String("path", l.path), zap.Error(err))
		return nil
	}

	var endpoints []observer.Endpoint
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		absPath := filepath.Join(l.path, entry.Name())
		endpoints = append(endpoints, observer.Endpoint{
			ID:      observer.EndpointID(absPath),
			Target:  absPath,
			Details: &TADirectory{Name: entry.Name(), Path: absPath},
		})
	}
	return endpoints
}
