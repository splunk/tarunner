package collector

import "go.opentelemetry.io/collector/component"

type host struct {
}

func (h host) GetExtensions() map[component.ID]component.Component {
	return nil
}
