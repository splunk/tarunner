// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package folderobserver

import "time"

// Config defines configuration for the folder observer extension.
type Config struct {
	// Path is the directory to watch for subdirectories. Each subdirectory
	// found will be emitted as a "folder" endpoint.
	Path string `mapstructure:"path"`

	// RefreshInterval controls how often the directory is re-scanned for
	// added or removed subdirectories.
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`
}
