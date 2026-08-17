// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package conf

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

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

// ReadInput parses an inputs.conf payload. appDir is the app directory
// (e.g. $SPLUNK_HOME/etc/apps/my_ta) used to resolve relative script paths
// to absolute paths at parse time. Pass an empty string to skip resolution.
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
		name := section.Name()
		if appDir != "" {
			name = resolveInputPath(name, appDir)
		}
		i := Input{
			Configuration: Configuration{
				Stanza: Stanza{
					Name:   name,
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

// resolveInputPath rewrites a stanza name's relative path component to an
// absolute path using appDir. Only script:// and bare (scripted) stanzas
// are rewritten; monitor://, tcp://, udp://, wineventlog:// are left as-is
// since their targets are not relative to the app directory.
func resolveInputPath(name, appDir string) string {
	switch {
	case hasScheme(name, "script"):
		rel := name[len("script://"):]
		return "script://" + absPath(appDir, rel)
	case hasScheme(name, "monitor"), hasScheme(name, "tcp"), hasScheme(name, "udp"), hasScheme(name, "wineventlog"), hasScheme(name, "batch"):
		return name
	default:
		// bare scripted input: name is the script filename
		abs := absPath(appDir, filepath.Join("bin", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH), name))
		return abs
	}
}

func hasScheme(name, scheme string) bool {
	prefix := scheme + "://"
	if len(name) < len(prefix) {
		return false
	}
	return strings.EqualFold(name[:len(prefix)], prefix)
}

func absPath(appDir, rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	abs, err := filepath.Abs(filepath.Join(appDir, rel))
	if err != nil {
		return filepath.Join(appDir, rel)
	}
	return abs
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
