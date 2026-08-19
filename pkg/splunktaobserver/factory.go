// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunktaobserver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

const defaultRefreshInterval = 60 * time.Second

func NewFactory() extension.Factory {
	return extension.NewFactory(
		component.MustNewType("splunk_ta_observer"),
		createDefaultConfig,
		createExtension,
		component.StabilityLevelDevelopment,
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		RefreshInterval: defaultRefreshInterval,
	}
}

func createExtension(_ context.Context, params extension.Settings, cfg component.Config) (extension.Extension, error) {
	return newObserver(params, cfg.(*Config))
}
