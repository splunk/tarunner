// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package conf

import (
	"errors"

	"gopkg.in/ini.v1"
)

// ErrNoHTTPOut is returned when outputs.conf contains no [httpout] stanza.
var ErrNoHTTPOut = errors.New("no [httpout] stanza found in outputs.conf")

// ConfMap is a parsed .conf file: stanza name -> key -> value.
type ConfMap map[string]map[string]string

// Output holds the settings from the [httpout] stanza in outputs.conf.
type Output struct {
	Token string
	URI   string
	// TODO: BatchSize and BatchTimeout from outputs.conf are not yet wired.
	// BatchSize maps to batcher MinSizeBytes, BatchTimeout to batcher FlushTimeout.
	// Both require configuring BatcherConfig on the exporter helper.
}

func ParseConf(payload []byte) (ConfMap, error) {
	f, err := ini.Load(payload)
	if err != nil {
		return nil, err
	}
	result := make(ConfMap)
	for _, section := range f.Sections() {
		name := section.Name()
		if name == ini.DefaultSection {
			continue
		}
		keys := make(map[string]string)
		for _, key := range section.Keys() {
			keys[key.Name()] = key.Value()
		}
		result[name] = keys
	}
	return result, nil
}

func MergeConf(layers []ConfMap) ConfMap {
	merged := make(ConfMap)
	for _, layer := range layers {
		for stanza, keys := range layer {
			if merged[stanza] == nil {
				merged[stanza] = make(map[string]string)
			}
			for k, v := range keys {
				merged[stanza][k] = v
			}
		}
	}
	return merged
}

func HTTPOut(merged ConfMap) (*Output, error) {
	keys, ok := merged["httpout"]
	if !ok {
		return nil, ErrNoHTTPOut
	}
	return &Output{
		Token: keys["httpEventCollectorToken"],
		URI:   keys["uri"],
	}, nil
}
