// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package conf

import (
	"encoding/xml"

	"gopkg.in/ini.v1"
)

const (
	appName        = "tarunner"
	xmlDeclaration = `<?xml version="1.0" encoding="UTF-8"?>
`
)

type Input struct {
	ServerHost    string        `xml:"server_host"`
	ServerURI     string        `xml:"server_uri"`
	SessionKey    string        `xml:"session_key"`
	CheckpointDir string        `xml:"checkpoint_dir"`
	Configuration Configuration `xml:"configuration"`
}

type Configuration struct {
	Stanza Stanza `xml:"stanza"`
}

type Params []Param

func (p Params) Get(name string) *Param {
	for _, param := range p {
		if param.Name == name {
			return &param
		}
	}
	return nil
}

type Stanza struct {
	Name   string `xml:"name,attr"`
	App    string `xml:"app,attr"`
	AppDir string `xml:"-"` // app directory used to resolve relative script paths at execution time
	Params Params `xml:"param"`
}

type Param struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",innerxml"`
}

// IsDisabled reports whether the stanza has disabled=1.
func (s *Stanza) IsDisabled() bool {
	p := s.Params.Get("disabled")
	return p != nil && p.Value == "1"
}

// ReadInput parses an inputs.conf payload. appDir is the app directory
// (e.g. $SPLUNK_HOME/etc/apps/my_ta) stored on each stanza for use when
// resolving relative script paths at execution time. Pass an empty string
// when the app directory is unknown or irrelevant.
func ReadInput(payload []byte, appDir string) ([]Input, error) {
	f, err := ini.Load(payload)
	if err != nil {
		return nil, err
	}
	result := make([]Input, len(f.Sections())-1)
	s := 0
	for _, section := range f.Sections() {
		if section.Name() == ini.DefaultSection {
			continue // disregard default section. We need a stanza per input.
		}
		i := Input{
			Configuration: Configuration{
				Stanza: Stanza{
					Name:   section.Name(),
					App:    appName,
					AppDir: appDir,
					Params: make([]Param, len(section.Keys())),
				},
			},
		}

		for keyIndex, key := range section.Keys() {
			i.Configuration.Stanza.Params[keyIndex] = Param{
				Name:  key.Name(),
				Value: key.Value(),
			}
		}

		result[s] = i
		s++
	}

	return result, nil
}

// MergeInputs merges multiple slices of inputs, with later slices taking
// precedence over earlier ones (local/ wins over default/).
// Stanzas are keyed by name; the last definition of a stanza wins entirely.
func MergeInputs(layers [][]Input) []Input {
	seen := make(map[string]int)
	var result []Input
	for _, layer := range layers {
		for _, input := range layer {
			name := input.Configuration.Stanza.Name
			if idx, ok := seen[name]; ok {
				result[idx] = input
			} else {
				seen[name] = len(result)
				result = append(result, input)
			}
		}
	}
	return result
}

func (i *Input) ToXML() ([]byte, error) {
	b, err := xml.MarshalIndent(i, "", "  ")
	return append([]byte(xmlDeclaration), b...), err
}
