// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package monitorreceiver

import (
	"github.com/splunk/tarunner/internal/conf"
)

type Config struct {
	Transform *conf.Transform `mapstructure:"-"`
	BaseDir   string          `mapstructure:"-"`
	Input     conf.Input      `mapstructure:"-"`
}
