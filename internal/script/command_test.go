// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package script

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/splunk/tarunner/internal/conf"
)

func TestDetermineCommandName(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		expected    string
		errExpected string
	}{
		{
			"script",
			"script://./bin/foo.sh",
			"bin/foo.sh",
			"",
		},
		{
			"outside the base dir",
			"script://../foo.sh",
			"",
			func() string {
				abs, _ := filepath.Abs("..")
				return fmt.Sprintf(`path "%s/foo.sh" is outside the base directory`, abs)
			}(),
		},
		{
			"modinput",
			"foo",
			func() string {
				return filepath.Join("bin", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH), "foo")
			}(),
			"",
		},
		{
			"invalid scheme",
			"invalid://foo",
			"",
			`unknown scheme "invalid"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := conf.Input{
				Configuration: conf.Configuration{
					Stanza: conf.Stanza{
						Name: test.command,
					},
				},
			}

			cmd, err := DetermineCommandName("", input)
			if test.errExpected != "" {
				require.Error(t, err)
				require.Equal(t, test.errExpected, err.Error())
			} else {
				abs, _ := filepath.Abs(test.expected)
				require.Equal(t, abs, cmd)
			}
		})
	}
}
