// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package scriptreceiver

import "github.com/splunk/tarunner/pkg/splunkta/conf"

type Config struct {
	conf.Input `mapstructure:"-"`
	BaseDir    string           `mapstructure:"-"`
	Props      []conf.Prop      `mapstructure:"-"`
	Transforms []conf.Transform `mapstructure:"-"`
}
