// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package monitorreceiver

import (
	"github.com/splunk/tarunner/pkg/splunkta/conf"
)

type Config struct {
	Input      conf.Input       `mapstructure:"-"`
	BaseDir    string           `mapstructure:"-"`
	Transforms []conf.Transform `mapstructure:"-"`
	Props      []conf.Prop      `mapstructure:"-"`
}
