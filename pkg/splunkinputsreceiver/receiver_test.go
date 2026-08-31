// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package splunkinputsreceiver

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTADirFromPath(t *testing.T) {
	appsDir := filepath.Join("/opt", "splunkforwarder", "etc", "apps")

	tests := []struct {
		name      string
		eventPath string
		want      string
	}{
		{
			name:      "file inside TA default dir",
			eventPath: filepath.Join(appsDir, "splunk_ta_syslog", "default", "inputs.conf"),
			want:      filepath.Join(appsDir, "splunk_ta_syslog"),
		},
		{
			name:      "file inside TA local dir",
			eventPath: filepath.Join(appsDir, "splunk_ta_syslog", "local", "inputs.conf"),
			want:      filepath.Join(appsDir, "splunk_ta_syslog"),
		},
		{
			name:      "appsDir itself",
			eventPath: appsDir,
			want:      "",
		},
		{
			name:      "deeply nested file",
			eventPath: filepath.Join(appsDir, "splunk_ta_syslog", "default", "subdir", "file.conf"),
			want:      filepath.Join(appsDir, "splunk_ta_syslog"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := taDirFromPath(tc.eventPath, appsDir)
			require.Equal(t, tc.want, got)
		})
	}
}
