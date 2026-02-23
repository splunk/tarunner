// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package conf

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestReadProps(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "props.conf"))
	require.NoError(t, err)
	props, err := ReadProps(b)
	require.NoError(t, err)
	require.Len(t, props, 1)
	require.Equal(t, "scoreboard", props[0].Name)
}
