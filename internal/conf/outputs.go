// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package conf

import (
	"strings"

	"gopkg.in/ini.v1"
)

// Output represents a single stanza from outputs.conf.
type Output struct {
	// Name is the stanza name, e.g. "httpout" or "httpout:primary".
	Name         string
	Token        string `xml:"httpEventCollectorToken"`
	URI          string `xml:"uri"`
	BatchSize    int    `xml:"batchSize"`
	BatchTimeout int    `xml:"batchTimeout"`
}

// IsHTTPOut reports whether this stanza is a httpout stanza.
func (o Output) IsHTTPOut() bool {
	return o.Name == "httpout" || strings.HasPrefix(o.Name, "httpout:")
}

// ReadOutputs parses an outputs.conf payload into a slice of Output.
func ReadOutputs(payload []byte) ([]Output, error) {
	f, err := ini.Load(payload)
	if err != nil {
		return nil, err
	}
	var result []Output
	for _, section := range f.Sections() {
		if section.Name() == ini.DefaultSection {
			continue
		}
		result = append(result, Output{
			Name:         section.Name(),
			Token:        section.Key("httpEventCollectorToken").String(),
			URI:          section.Key("uri").String(),
			BatchSize:    section.Key("batchSize").MustInt(),
			BatchTimeout: section.Key("batchTimeout").MustInt(),
		})
	}
	return result, nil
}
