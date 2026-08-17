// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package conf

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfAndHTTPOut(t *testing.T) {
	payload, err := os.ReadFile("testdata/outputs.conf")
	require.NoError(t, err)

	parsed, err := ParseConf(payload)
	require.NoError(t, err)

	output, err := HTTPOut(parsed)
	require.NoError(t, err)
	require.NotNil(t, output)

	assert.Equal(t, "token-default", output.Token)
	assert.Equal(t, "https://splunk:8088/services/collector/event", output.URI)
}

func TestHTTPOutMissing(t *testing.T) {
	parsed, err := ParseConf([]byte("[tcpout]\nserver = splunk:9997\n"))
	require.NoError(t, err)
	_, err = HTTPOut(parsed)
	require.ErrorIs(t, err, ErrNoHTTPOut)
}

func TestMergeConf(t *testing.T) {
	base, err := ParseConf([]byte("[httpout]\nhttpEventCollectorToken = base-token\nuri = https://base:8088/services/collector/event\n"))
	require.NoError(t, err)

	override, err := ParseConf([]byte("[httpout]\nhttpEventCollectorToken = override-token\n"))
	require.NoError(t, err)

	merged := MergeConf([]map[string]map[string]string{base, override})
	output, err := HTTPOut(merged)
	require.NoError(t, err)

	// override wins for token, base value kept for uri
	assert.Equal(t, "override-token", output.Token)
	assert.Equal(t, "https://base:8088/services/collector/event", output.URI)
}
