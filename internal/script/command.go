// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package script

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/splunk/tarunner/internal/conf"
	"github.com/splunk/tarunner/internal/stanza"
)

func DetermineCommandName(baseDir string, input conf.Input) (string, error) {
	parsed, err := stanza.ParseName(input.Configuration.Stanza.Name)
	if err != nil {
		return "", err
	}
	// Use AppDir from the stanza when available; it points to the specific app
	// directory the conf was parsed from, which is more accurate than baseDir
	// when multiple apps are layered together.
	resolveDir := baseDir
	if input.Configuration.Stanza.AppDir != "" {
		resolveDir = input.Configuration.Stanza.AppDir
	}
	switch parsed.Kind {
	case "monitor", "batch":
		return parsed.Target, nil
	case "script":
		if filepath.IsAbs(parsed.Target) {
			return parsed.Target, nil
		}
		return GetPath(resolveDir, parsed.Target)
	case "":
		if filepath.IsAbs(parsed.Target) {
			return parsed.Target, nil
		}
		return GetPath(resolveDir, filepath.Join("bin", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH), parsed.Target))
	default:
		return "", fmt.Errorf("unknown scheme %q", parsed.Kind)
	}
}

func GetPath(baseDir, path string) (string, error) {
	var resolvedPath string
	if filepath.IsAbs(path) {
		resolvedPath = path
	} else {
		var err error
		resolvedPath, err = filepath.Abs(filepath.Join(baseDir, path))
		if err != nil {
			return "", err
		}
	}
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}

	relPath, err := filepath.Rel(absBaseDir, resolvedPath)
	if err != nil {
		return "", err
	}
	if relPath == "." || strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path '%s' is outside the base directory", filepath.Clean(resolvedPath))
	}

	return resolvedPath, nil
}
