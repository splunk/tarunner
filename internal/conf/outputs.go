// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package conf

import (
	"errors"

	"gopkg.in/ini.v1"
)

// ErrNoHTTPOut is returned when outputs.conf contains no [httpout] stanza.
var ErrNoHTTPOut = errors.New("no [httpout] stanza found in outputs.conf")

// Output holds the settings from the [httpout] stanza in outputs.conf.
type Output struct {
	Token string
	URI   string
	// TODO: BatchSize and BatchTimeout from outputs.conf are not yet wired.
	// BatchSize maps to batcher MinSizeBytes, BatchTimeout to batcher FlushTimeout.
	// Both require configuring BatcherConfig on the exporter helper.
}

// ParseConf parses a single outputs.conf payload into a generic
// stanza -> key -> value map. All stanza names and keys are lower-cased.
func ParseConf(payload []byte) (map[string]map[string]string, error) {
	f, err := ini.Load(payload)
	if err != nil {
		return nil, err
	}
	result := make(map[string]map[string]string)
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

// MergeConf merges multiple parsed outputs.conf layers in order (lowest to
// highest precedence). Later layers override keys from earlier ones.
func MergeConf(layers []map[string]map[string]string) map[string]map[string]string {
	merged := make(map[string]map[string]string)
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

// HTTPOut extracts the [httpout] stanza from a merged conf map.
// Returns ErrNoHTTPOut if the stanza is absent.
func HTTPOut(merged map[string]map[string]string) (*Output, error) {
	keys, ok := merged["httpout"]
	if !ok {
		return nil, ErrNoHTTPOut
	}
	return &Output{
		Token: keys["httpEventCollectorToken"],
		URI:   keys["uri"],
	}, nil
}
