// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package filter provides helpers for translating Splunk whitelist/blacklist
// params into filelog Include/Exclude globs and stanza filter operators.
package filter

import (
	"fmt"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/filter"
)

// IsGlobPattern reports whether s is a glob pattern suitable for filelog's
// Include/Exclude fields. Splunk whitelist/blacklist values can be either
// glob patterns (containing *, ?, or [) or PCRE regexes (containing (, |,
// $, or \). The latter are not valid globs and must not be passed to filelog.
func IsGlobPattern(s string) bool {
	return s != "" && strings.ContainsAny(s, "*?[") && !strings.ContainsAny(s, "(|$\\")
}

// IsPCREPattern reports whether s looks like a PCRE regex — i.e. contains
// characters that are meaningful in PCRE but not in glob patterns.
func IsPCREPattern(s string) bool {
	return s != "" && strings.ContainsAny(s, "(|$\\.+?^")
}

// NewWhitelistOperator returns a filter operator that drops entries whose
// log.file.path does NOT match the given PCRE regex.
func NewWhitelistOperator(regex string) operator.Config {
	c := filter.NewConfigWithID("whitelist-filter")
	c.Expression = fmt.Sprintf(`!(attributes["log.file.path"] matches %q)`, regex)
	return operator.NewConfig(c)
}

// NewBlacklistOperator returns a filter operator that drops entries whose
// log.file.path matches the given PCRE regex.
func NewBlacklistOperator(regex string) operator.Config {
	c := filter.NewConfigWithID("blacklist-filter")
	c.Expression = fmt.Sprintf(`attributes["log.file.path"] matches %q`, regex)
	return operator.NewConfig(c)
}
