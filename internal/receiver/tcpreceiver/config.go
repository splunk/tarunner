// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package tcpreceiver

import (
	"github.com/splunk/tarunner/internal/conf"
	"github.com/splunk/tarunner/internal/stanza"
)

type Config struct {
	Transforms []conf.Transform `mapstructure:"-"`
	Props      []conf.Prop      `mapstructure:"-"`

	BaseDir string     `mapstructure:"-"`
	Input   conf.Input `mapstructure:"-"`
}

func (cfg *Config) Validate() error {
	_, err := stanza.ParseName(cfg.Input.Configuration.Stanza.Name)
	return err
}
