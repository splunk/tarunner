// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package tcpreceiver

import (
	"net/url"

	"github.com/splunk/tarunner/internal/conf"
)

type Config struct {
	Transforms []conf.Transform `mapstructure:"-"`
	Props      []conf.Prop      `mapstructure:"-"`

	BaseDir string     `mapstructure:"-"`
	Input   conf.Input `mapstructure:"-"`
}

func (cfg *Config) Validate() error {
	_, err := url.Parse(cfg.Input.Configuration.Stanza.Name)
	return err
}
