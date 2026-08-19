// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

type Config struct {
	// BaseDir is the Splunk installation root ($SPLUNK_HOME). inputs.conf,
	// transforms.conf, and props.conf are discovered across etc/apps/* using
	// standard Splunk precedence. Overrides $SPLUNK_HOME when set.
	// Mutually exclusive with Path.
	BaseDir string `mapstructure:"base_dir"`

	// Path is the absolute path to a single TA directory. When set, only that
	// directory's inputs.conf, transforms.conf, and props.conf are read with no
	// btool-style layering. Intended for use with the splunk_ta_observer via
	// receiver_creator. Mutually exclusive with BaseDir.
	Path string `mapstructure:"path"`
}
