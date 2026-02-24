// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package scriptreceiver

import "github.com/splunk/tarunner/internal/conf"

type Config struct {
	BaseDir    string           `mapstructure:"-"`
	Props      []conf.Prop      `mapstructure:"-"`
	Transforms []conf.Transform `mapstructure:"-"`
	conf.Input `mapstructure:"-"`
}
