// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

type Config struct {
	// BaseDir is the Splunk installation root ($SPLUNK_HOME). inputs.conf,
	// transforms.conf, and props.conf are discovered across etc/apps/* using
	// standard Splunk precedence. Overrides $SPLUNK_HOME when set.
	BaseDir string `mapstructure:"base_dir"`
}
