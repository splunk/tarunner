// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package monitorreceiver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/file"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/filter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/move"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/noop"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/split"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/trim"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"

	"github.com/splunk/tarunner/pkg/splunkta/operator/prop"
	"github.com/splunk/tarunner/pkg/splunkta/script"
)

type monitor struct {
	logger *zap.Logger
}

// Type is the receiver type
func (monitor) Type() component.Type {
	return component.MustNewType("monitor")
}

// CreateDefaultConfig creates a config with type and version
func (monitor) CreateDefaultConfig() component.Config {
	return createDefaultConfig()
}

func createDefaultConfig() *Config {
	return &Config{}
}

// BaseConfig gets the base config from config
func (monitor) BaseConfig(cfg component.Config) adapter.BaseConfig {
	rcfg := cfg.(Config)
	var operators []operator.Config

	// Insert PCRE whitelist/blacklist filters before any other processing.
	// The log.file.path attribute is set by filelog and available here.
	if w := rcfg.Input.Configuration.Stanza.Params.Get("whitelist"); w != nil && isPCREPattern(w.Value) {
		operators = append(operators, createWhitelistFilterOperator(w.Value))
	}
	if b := rcfg.Input.Configuration.Stanza.Params.Get("blacklist"); b != nil && isPCREPattern(b.Value) {
		operators = append(operators, createBlacklistFilterOperator(b.Value))
	}

	operators = append(operators, createSetSourceOperator())

	for _, p := range rcfg.Props {
		ops := prop.CreateOperatorConfigs(p, rcfg.Transforms)
		operators = append(operators, ops...)
	}

	endNoop := noop.NewConfigWithID("end")

	metadata := renameMetadata()
	endNoop.OutputIDs = []string{metadata[0].ID()}
	operators = append(operators, operator.NewConfig(endNoop))
	operators = append(operators, metadata...)

	return adapter.BaseConfig{
		Operators: operators,
	}
}

// createWhitelistFilterOperator drops entries whose log.file.path does NOT match the regex.
// filter drops entries when expression is true — drop if path does NOT match whitelist.
func createWhitelistFilterOperator(regex string) operator.Config {
	c := filter.NewConfigWithID("whitelist-filter")
	c.Expression = fmt.Sprintf(`!(attributes["log.file.path"] matches %q)`, regex)
	return operator.NewConfig(c)
}

// createBlacklistFilterOperator drops entries whose log.file.path matches the regex.
// filter drops entries when expression is true — drop if path matches blacklist.
func createBlacklistFilterOperator(regex string) operator.Config {
	c := filter.NewConfigWithID("blacklist-filter")
	c.Expression = fmt.Sprintf(`attributes["log.file.path"] matches %q`, regex)
	return operator.NewConfig(c)
}

func createSetSourceOperator() operator.Config {
	c := move.NewConfigWithID("start")
	c.From = entry.NewAttributeField("log.file.path")
	c.To = entry.NewAttributeField("source")
	c.OnError = "send_quiet"
	return operator.NewConfig(c)
}

func (t monitor) InputConfig(config component.Config) operator.Config {
	rcfg := config.(Config)
	oc := file.NewConfig()
	path, err := script.DetermineCommandName(rcfg.BaseDir, rcfg.Input)
	if err != nil {
		t.logger.Error("error reading command", zap.Error(err))
		return operator.NewConfig(oc)
	}
	// pathIsGlob reports whether the path already contains glob metacharacters
	// (e.g. monitor:///home/*/.bash_history). In that case filelog can use it
	// directly without further expansion.
	pathIsGlob := strings.ContainsAny(path, "*?[")

	w := rcfg.Input.Configuration.Stanza.Params.Get("whitelist")
	var allowlist string
	switch {
	case pathIsGlob:
		// Path is already a glob pattern — use it as-is; whitelist is ignored
		// because there is no sensible directory to join it against.
		allowlist = path
	case w != nil && isGlobPattern(w.Value):
		// whitelist is a glob pattern relative to the monitored directory
		// (e.g. "*.log").
		allowlist = filepath.Join(path, w.Value)
	case w != nil:
		// whitelist param is present but either empty or a Splunk regex (which
		// is not a valid filelog glob). In both cases match all files directly
		// under the directory.
		allowlist = filepath.Join(path, "*")
	default:
		// No whitelist param: if the path is a directory, expand to dir/* so
		// filelog can match files inside it. If it's a specific file (or the
		// path doesn't exist yet), use it as-is.
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			allowlist = filepath.Join(path, "*")
		} else {
			allowlist = path
		}
	}
	oc.Include = []string{allowlist}
	t.logger.Info("monitor receiver include pattern",
		zap.String("stanza", rcfg.Input.Configuration.Stanza.Name),
		zap.String("path", path),
		zap.String("include", allowlist),
	)
	if b := rcfg.Input.Configuration.Stanza.Params.Get("blacklist"); b != nil && isGlobPattern(b.Value) {
		oc.Exclude = []string{filepath.Join(path, b.Value)}
		t.logger.Info("monitor receiver exclude pattern",
			zap.String("stanza", rcfg.Input.Configuration.Stanza.Name),
			zap.String("exclude", oc.Exclude[0]),
		)
	}
	if hostParam := rcfg.Input.Configuration.Stanza.Params.Get("host"); hostParam != nil {
		// TODO: find a way to run host detection when requested.
		oc.Attributes["host"] = helper.ExprStringConfig(hostParam.Value)
	}

	if indexParam := rcfg.Input.Configuration.Stanza.Params.Get("index"); indexParam != nil {
		oc.Attributes["index"] = helper.ExprStringConfig(indexParam.Value)
	}

	if sourceTypeParam := rcfg.Input.Configuration.Stanza.Params.Get("sourcetype"); sourceTypeParam != nil {
		oc.Attributes["sourcetype"] = helper.ExprStringConfig(sourceTypeParam.Value)
	}

	if sourceParam := rcfg.Input.Configuration.Stanza.Params.Get("source"); sourceParam != nil {
		oc.Attributes["source"] = helper.ExprStringConfig(sourceParam.Value)
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

	return operator.NewConfig(oc)
}

// isGlobPattern reports whether s is a glob pattern suitable for filelog's
// Include/Exclude fields. Splunk whitelist/blacklist values can be either
// glob patterns (containing *, ?, or [) or PCRE regexes (containing (, |,
// $, or \). The latter are not valid globs and must not be passed to filelog.
func isGlobPattern(s string) bool {
	return s != "" && strings.ContainsAny(s, "*?[") && !strings.ContainsAny(s, "(|$\\")
}

// isPCREPattern reports whether s looks like a PCRE regex — i.e. contains
// characters that are meaningful in PCRE but not in glob patterns.
func isPCREPattern(s string) bool {
	return s != "" && strings.ContainsAny(s, "(|$\\.+?^")
}

func renameMetadata() []operator.Config {
	source := move.NewConfigWithID("end-source")
	source.From = entry.NewAttributeField("source")
	source.To = entry.NewAttributeField("com.splunk.source")
	source.OnError = "send_quiet"
	source.OutputIDs = []string{"end-sourcetype"}

	sourceType := move.NewConfigWithID("end-sourcetype")
	sourceType.From = entry.NewAttributeField("sourcetype")
	sourceType.To = entry.NewAttributeField("com.splunk.sourcetype")
	sourceType.OnError = "send_quiet"
	sourceType.OutputIDs = []string{"end-host"}

	host := move.NewConfigWithID("end-host")
	host.From = entry.NewAttributeField("host")
	host.To = entry.NewAttributeField("host.name")
	host.OnError = "send_quiet"
	host.OutputIDs = []string{"end-index"}

	index := move.NewConfigWithID("end-index")
	index.From = entry.NewAttributeField("index")
	index.To = entry.NewAttributeField("com.splunk.index")
	index.OnError = "send_quiet"

	return []operator.Config{
		operator.NewConfig(source),
		operator.NewConfig(sourceType),
		operator.NewConfig(host),
		operator.NewConfig(index),
	}
}
