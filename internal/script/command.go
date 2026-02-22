// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package script

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/splunk/tarunner/internal/conf"
)

func DetermineCommandName(baseDir string, input conf.Input) (string, error) {
	parsed, err := url.Parse(input.Configuration.Stanza.Name)
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "script":
		return GetPath(baseDir, filepath.Join(parsed.Host, parsed.Path))
	case "":
		return GetPath(baseDir, filepath.Join("bin", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH), input.Configuration.Stanza.Name))
	default:
		return "", fmt.Errorf("unknown scheme %q", parsed.Scheme)
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
		return "", fmt.Errorf("path %q is outside the base directory", resolvedPath)
	}

	return resolvedPath, nil
}
