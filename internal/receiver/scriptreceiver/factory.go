// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package scriptreceiver

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"go.opentelemetry.io/collector/receiver"

	"github.com/splunk/tarunner/internal/receiver/scriptreceiver/internal/metadata"
)

func NewFactory() receiver.Factory {
	return adapter.NewFactory(scriptReceiver{}, metadata.LogsStability)
}
