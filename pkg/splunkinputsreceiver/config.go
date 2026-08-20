// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

import "go.opentelemetry.io/collector/component"

type Config struct {
	// BaseDir is the Splunk installation root ($SPLUNK_HOME). inputs.conf,
	// transforms.conf, and props.conf are discovered across etc/apps/* using
	// standard Splunk precedence. Overrides $SPLUNK_HOME when set.
	// Mutually exclusive with Path and WatchObservers.
	BaseDir string `mapstructure:"base_dir"`

	// Path is the absolute path to a single TA directory. When set, only that
	// directory's inputs.conf, transforms.conf, and props.conf are read with no
	// btool-style layering. Mutually exclusive with BaseDir and WatchObservers.
	Path string `mapstructure:"path"`

	// WatchObservers is a list of observer extension IDs (e.g. folder_observer)
	// to subscribe to. For each directory endpoint received, a splunk_inputs
	// sub-receiver is started using that directory as Path. Mutually exclusive
	// with BaseDir and Path.
	WatchObservers []component.ID `mapstructure:"watch_observers"`
}
