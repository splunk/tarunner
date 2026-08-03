// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkoutputsexporter

// Config holds the configuration for the splunk_outputs exporter.
type Config struct {
	// BaseDir is the directory containing local/ or default/ subdirectories
	// with outputs.conf. Defaults to $SPLUNK_HOME if empty.
	BaseDir string `mapstructure:"base_dir"`
}
