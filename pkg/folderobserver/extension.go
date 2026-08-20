// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package folderobserver

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

// folderDetails holds details about a discovered directory endpoint.
type folderDetails struct {
	Name string
	Path string
}

func (f *folderDetails) Type() observer.EndpointType { return observer.EndpointType("folder") }
func (f *folderDetails) Env() observer.EndpointEnv {
	return observer.EndpointEnv{"name": f.Name, "path": f.Path}
}

type folderObserver struct {
	*endpointswatcher.EndpointsWatcher
}

var _ extension.Extension = (*folderObserver)(nil)

func newObserver(params extension.Settings, config *Config) (extension.Extension, error) {
	return &folderObserver{
		EndpointsWatcher: endpointswatcher.New(
			&folderLister{path: config.Path, logger: params.Logger},
			config.RefreshInterval,
			params.Logger,
		),
	}, nil
}

func (*folderObserver) Start(context.Context, component.Host) error { return nil }

func (o *folderObserver) Shutdown(context.Context) error {
	o.StopListAndWatch()
	return nil
}

type folderLister struct {
	path   string
	logger *zap.Logger
}

func (l *folderLister) ListEndpoints() []observer.Endpoint {
	entries, err := os.ReadDir(l.path)
	if err != nil {
		l.logger.Error("folder_observer: failed to read directory", zap.String("path", l.path), zap.Error(err))
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
			Details: &folderDetails{Name: entry.Name(), Path: absPath},
		})
	}
	return endpoints
}
