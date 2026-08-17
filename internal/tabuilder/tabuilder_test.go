// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package tabuilder

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk/tarunner/internal/conf"
)

func TestReadOutputs(t *testing.T) {
	rootDir := filepath.Join("testdata", "outputs")
	tests := []struct {
		name          string
		splunkHome    string
		expectedToken string
		expectedURI   string
		expectedErr   error
	}{
		{
			name:          "system_default_only",
			splunkHome:    filepath.Join(rootDir, "system_default"),
			expectedToken: "token-system-default",
			expectedURI:   "https://system-default:8088/services/collector/event",
		},
		{
			name:          "app_overrides_system_default",
			splunkHome:    filepath.Join(rootDir, "app_overrides"),
			expectedToken: "token-app-default",
			expectedURI:   "https://app-local:8088/services/collector/event",
		},
		{
			name:          "system_local_wins_over_app",
			splunkHome:    filepath.Join(rootDir, "system_local_wins"),
			expectedToken: "token-system-local",
			expectedURI:   "https://system-local:8088/services/collector/event",
		},
		{
			name:        "no_httpout",
			splunkHome:  filepath.Join(rootDir, "no_httpout"),
			expectedErr: conf.ErrNoHTTPOut,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			merged, err := ReadOutputs(test.splunkHome)
			if test.expectedErr != nil {
				require.NoError(t, err) // ReadOutputs itself doesn't error on missing stanza
				_, httpErr := HTTPOut(merged)
				require.ErrorIs(t, httpErr, test.expectedErr)
				return
			}
			require.NoError(t, err)
			output, err := HTTPOut(merged)
			require.NoError(t, err)
			require.NotNil(t, output)
			assert.Equal(t, test.expectedToken, output.Token)
			assert.Equal(t, test.expectedURI, output.URI)
		})
	}
}

func TestReadTransforms(t *testing.T) {
	rootDir := filepath.Join("testdata", "transforms")
	tests := []struct {
		name          string
		splunkHome    string
		expectedName  string
		expectedRegex string
	}{
		{
			name:          "default",
			splunkHome:    filepath.Join(rootDir, "splunk_default"),
			expectedName:  "example_default",
			expectedRegex: "default",
		},
		{
			name:          "local",
			splunkHome:    filepath.Join(rootDir, "splunk_local"),
			expectedName:  "example_local",
			expectedRegex: "local",
		},
		{
			name:          "both",
			splunkHome:    filepath.Join(rootDir, "splunk_both"),
			expectedName:  "example_transform",
			expectedRegex: "local",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transforms, err := ReadTransforms(test.splunkHome)
			require.NoError(t, err)
			require.Len(t, transforms, 1)
			require.Equal(t, test.expectedName, transforms[0].Name)
			require.Equal(t, test.expectedRegex, transforms[0].Regex)
		})
	}
}
