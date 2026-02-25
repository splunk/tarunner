// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package monitorreceiver

import (
	"github.com/splunk/tarunner/internal/conf"
)

type Config struct {
	Transforms []conf.Transform `mapstructure:"-"`
	Props      []conf.Prop      `mapstructure:"-"`

	BaseDir string     `mapstructure:"-"`
	Input   conf.Input `mapstructure:"-"`
}
