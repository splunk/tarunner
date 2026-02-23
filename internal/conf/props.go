// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package conf

import "gopkg.in/ini.v1"

type Prop struct {
	Name                  string
	TimePrefix            string
	TimeFormat            string
	MaxTimestampLookAhead int
	DatetimeConfig        string
	NoBinaryCheck         bool
	Category              string
}

func ReadProps(payload []byte) ([]Prop, error) {
	f, err := ini.Load(payload)
	if err != nil {
		return nil, err
	}
	result := make([]Prop, len(f.Sections())-1)
	s := 0
	for _, section := range f.Sections() {
		if section.Name() == ini.DefaultSection {
			continue // disregard default section. We need a stanza per transform.
		}
		maxTimestampLookAhead, err := section.Key("MAX_TIMESTAMP_LOOKAHEAD").Int()
		if err != nil {
			return nil, err
		}
		noBinaryCheck, err := section.Key("NO_BINARY_CHECK").Bool()
		if err != nil {
			return nil, err
		}
		p := Prop{
			Name:                  section.Name(),
			TimePrefix:            section.Key("TIME_PREFIX").String(),
			TimeFormat:            section.Key("TIME_FORMAT").String(),
			MaxTimestampLookAhead: maxTimestampLookAhead,
			DatetimeConfig:        section.Key("DATETIME_CONFIG").String(),
			NoBinaryCheck:         noBinaryCheck,
			Category:              section.Key("category").String(),
		}

		result[s] = p
		s++
	}

	return result, nil
}
