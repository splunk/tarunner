// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package monitorreceiver

import (
	"github.com/splunk/tarunner/internal/conf"
)

type Config struct {
	BaseDir   string          `mapstructure:"base_dir"`
	Input     conf.Input      `mapstructure:"input"`
	Transform *conf.Transform `mapstructure:"transform"`
}
