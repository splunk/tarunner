// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package monitorreceiver

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/splunk/tarunner/internal/monitorreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver"
)

func NewFactory() receiver.Factory {
	return adapter.NewFactory(receiverType{}, metadata.LogsStability)
}
