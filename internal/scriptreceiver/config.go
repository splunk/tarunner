// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package scriptreceiver

import "github.com/splunk/tarunner/internal/conf"

type Config struct {
	Transform *conf.Transform `mapstructure:"transform"`
	BaseDir   string          `mapstructure:"base_dir"`
	Input     conf.Input      `mapstructure:"input"`
}
