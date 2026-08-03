// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package conf

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadOutputs(t *testing.T) {
	payload, err := os.ReadFile("testdata/outputs.conf")
	require.NoError(t, err)

	outputs, err := ReadOutputs(payload)
	require.NoError(t, err)
	require.Len(t, outputs, 3)

	assert.Equal(t, "httpout", outputs[0].Name)
	assert.Equal(t, "token-default", outputs[0].Token)
	assert.Equal(t, "https://splunk:8088/services/collector/event", outputs[0].URI)
	assert.Equal(t, 32768, outputs[0].BatchSize)
	assert.Equal(t, 10, outputs[0].BatchTimeout)
	assert.True(t, outputs[0].IsHTTPOut())

	assert.Equal(t, "httpout:secondary", outputs[1].Name)
	assert.Equal(t, "token-secondary", outputs[1].Token)
	assert.Equal(t, "https://splunk2:8088/services/collector/event", outputs[1].URI)
	assert.Equal(t, 0, outputs[1].BatchSize)
	assert.Equal(t, 0, outputs[1].BatchTimeout)
	assert.True(t, outputs[1].IsHTTPOut())

	assert.Equal(t, "tcpout", outputs[2].Name)
	assert.False(t, outputs[2].IsHTTPOut())
}
