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

	output, err := ReadOutputs(payload)
	require.NoError(t, err)
	require.NotNil(t, output)

	assert.Equal(t, "token-default", output.Token)
	assert.Equal(t, "https://splunk:8088/services/collector/event", output.URI)
}

func TestReadOutputsNoHTTPOut(t *testing.T) {
	output, err := ReadOutputs([]byte("[tcpout]\nserver = splunk:9997\n"))
	require.NoError(t, err)
	assert.Nil(t, output)
}
