// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package tabuilder

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadTransforms(t *testing.T) {
	rootDir := filepath.Join("testdata", "transforms")
	tests := []struct {
		name          string
		path          string
		expectedName  string
		expectedRegex string
	}{
		{
			name:          "default",
			path:          filepath.Join(rootDir, "default"),
			expectedName:  "example_default",
			expectedRegex: "default",
		},
		{
			name:          "local",
			path:          filepath.Join(rootDir, "local"),
			expectedName:  "example_local",
			expectedRegex: "local",
		},
		{
			name:          "both",
			path:          filepath.Join(rootDir, "both"),
			expectedName:  "example_local2",
			expectedRegex: "local",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transforms, err := ReadTransforms(test.path)
			require.NoError(t, err)
			require.Len(t, transforms, 1)
			require.Equal(t, test.expectedName, transforms[0].Name)
			require.Equal(t, test.expectedRegex, transforms[0].Regex)
		})
	}
}
