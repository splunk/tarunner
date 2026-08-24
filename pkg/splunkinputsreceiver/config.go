// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

type Config struct {
	// Path is the absolute path to a single TA directory
	// (e.g. /opt/splunk/etc/apps/Splunk_TA_nix).
	// The Splunk home is derived automatically by walking up the directory tree.
	Path string `mapstructure:"path"`
}
