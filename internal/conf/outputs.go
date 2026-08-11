// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package conf

import (
	"gopkg.in/ini.v1"
)

// Output holds the settings from the [httpout] stanza in outputs.conf.
// outputs.conf supports exactly one [httpout] stanza.
type Output struct {
	Token string
	URI   string
	// TODO: BatchSize and BatchTimeout from outputs.conf are not yet wired.
	// BatchSize maps to batcher MinSizeBytes, BatchTimeout to batcher FlushTimeout.
	// Both require configuring BatcherConfig on the exporter helper.
}

// ReadOutputs parses an outputs.conf payload and returns the [httpout] stanza,
// or nil if no [httpout] stanza is present.
func ReadOutputs(payload []byte) (*Output, error) {
	f, err := ini.Load(payload)
	if err != nil {
		return nil, err
	}
	section, err := f.GetSection("httpout")
	if err != nil {
		// Section not found — ini returns an error in this case.
		return nil, nil
	}
	return &Output{
		Token: section.Key("httpEventCollectorToken").String(),
		URI:   section.Key("uri").String(),
	}, nil
}
