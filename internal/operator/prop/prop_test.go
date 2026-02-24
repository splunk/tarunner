// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prop

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/copy"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/noop"

	"github.com/stretchr/testify/require"

	"github.com/splunk/tarunner/internal/conf"
)

func TestCreateProps(t *testing.T) {
	ops := CreateOperatorConfigs(conf.Prop{
		Name: "foo",
		FieldAliases: []conf.FieldAlias{
			{
				Name: "foo",
				From: "foo",
				To:   "bar",
			},
			{
				Name: "foobar",
				From: "foo",
				To:   "foobar",
			},
		},
	}, nil)

	require.Len(t, ops, 4)
	require.Equal(t, "foo-start", ops[0].Builder.(*noop.Config).OperatorID)
	require.Equal(t, "foo-copy", ops[1].Builder.(*copy.Config).OperatorID)
	require.Equal(t, "foobar-copy", ops[2].Builder.(*copy.Config).OperatorID)
	require.Equal(t, `attributes.foobar`, ops[2].Builder.(*copy.Config).To.String())
	require.Equal(t, []string{"foo-end"}, ops[2].Builder.(*copy.Config).OutputIDs)
	require.Equal(t, "foo-end", ops[3].Builder.(*noop.Config).OperatorID)
}
