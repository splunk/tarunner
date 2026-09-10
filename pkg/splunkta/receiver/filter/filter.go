// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package filter provides helpers for translating Splunk whitelist/blacklist
// params into filelog Include/Exclude globs and stanza filter operators.
package filter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/file"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/filter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/split"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/trim"
	"go.uber.org/zap"

	"github.com/splunk/tarunner/pkg/splunkta/conf"
)

// IsGlobPattern reports whether s is a glob pattern suitable for filelog's
// Include/Exclude fields. Splunk whitelist/blacklist values can be either
// glob patterns (containing *, ?, or [) or PCRE regexes. A value is treated
// as a glob only when it contains glob metacharacters and none of the
// characters that are meaningful in PCRE but not in globs.
func IsGlobPattern(s string) bool {
	return s != "" && strings.ContainsAny(s, "*?[") && !strings.ContainsAny(s, "(|$\\.+^")
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

// ApplyIncludeExclude sets oc.Include and oc.Exclude based on the Splunk
// whitelist/blacklist params and the resolved path. It also logs the resulting
// patterns at debug level using the provided receiver name as context.
func ApplyIncludeExclude(oc *file.Config, path string, stanza conf.Stanza, receiverName string, logger *zap.Logger) {
	w := stanza.Params.Get("whitelist")
	var allowlist string
	switch {
	case IsGlobPattern(path):
		allowlist = path
	case w != nil && IsGlobPattern(w.Value):
		allowlist = filepath.Join(path, w.Value)
	case w != nil:
		// whitelist param is present but either empty or a PCRE regex (not a valid
		// filelog glob). Expand to dir/* so filelog picks up all files; the PCRE
		// regex is applied as a filter operator in BaseConfig.
		allowlist = filepath.Join(path, "*")
	default:
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			allowlist = filepath.Join(path, "*")
		} else {
			allowlist = path
		}
	}
	oc.Include = []string{allowlist}
	logger.Debug(
		receiverName+" receiver include pattern",
		zap.String("stanza", stanza.Name),
		zap.String("path", path),
		zap.String("include", allowlist),
	)
	if b := stanza.Params.Get("blacklist"); b != nil && IsGlobPattern(b.Value) {
		oc.Exclude = []string{filepath.Join(path, b.Value)}
		logger.Debug(
			receiverName+" receiver exclude pattern",
			zap.String("stanza", stanza.Name),
			zap.String("exclude", oc.Exclude[0]),
		)
	}
}

// ApplyStanzaConfig sets file.Config attributes and defaults that are common
// to both monitor and batch receivers.
func ApplyStanzaConfig(oc *file.Config, stanza conf.Stanza) {
	for _, name := range []string{"host", "index", "sourcetype", "source"} {
		if p := stanza.Params.Get(name); p != nil {
			oc.Attributes[name] = helper.ExprStringConfig(p.Value)
		}
	}
	oc.IncludeFilePath = true
	oc.Encoding = "utf-8"
	oc.StartAt = "beginning"
	oc.SplitConfig = split.Config{
		LineStartPattern: "^",
	}
	oc.TrimConfig = trim.Config{
		PreserveLeading:  true,
		PreserveTrailing: true,
	}
}
