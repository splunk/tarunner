package conf

import "gopkg.in/ini.v1"

type Transform struct {
	Name   string
	Regex  string
	Format string
}

func ReadTransforms(payload []byte) ([]Transform, error) {
	f, err := ini.Load(payload)
	if err != nil {
		return nil, err
	}
	result := make([]Transform, len(f.Sections())-1)
	s := 0
	for _, section := range f.Sections() {
		if section.Name() == ini.DefaultSection {
			continue // disregard default section. We need a stanza per transform.
		}
		t := Transform{
			Name:   section.Name(),
			Regex:  section.Key("REGEX").String(),
			Format: section.Key("FORMAT").String(),
		}

		result[s] = t
		s++
	}

	return result, nil
}
