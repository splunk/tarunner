// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunktaobserver

import "time"

// Config defines configuration for the Splunk TA observer.
type Config struct {
	// Path is the directory to scan for TA subdirectories, typically
	// $SPLUNK_HOME/etc/apps.
	Path string `mapstructure:"path"`

	// RefreshInterval controls how often the directory is re-scanned for
	// added or removed TAs.
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`
}
