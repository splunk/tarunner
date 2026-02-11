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
	Params Params `xml:"param"`
}

type Param struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",innerxml"`
}

func ReadInput(payload []byte) ([]Input, error) {
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

func (i *Input) ToXML() ([]byte, error) {
	b, err := xml.MarshalIndent(i, "", "  ")
	return append([]byte(xmlDeclaration), b...), err
}
