// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build tools

package tools

// based on https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/internal/tools/tools.go
import (
	_ "github.com/client9/misspell/cmd/misspell"
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "github.com/google/addlicense"
	_ "go.opentelemetry.io/build-tools/chloggen"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment"
	_ "mvdan.cc/gofumpt"
)
