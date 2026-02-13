// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package collector

import "go.opentelemetry.io/collector/component"

type host struct{}

func (h host) GetExtensions() map[component.ID]component.Component {
	return nil
}
