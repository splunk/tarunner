// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prop

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/recombine"
	"go.opentelemetry.io/collector/featuregate"

	"github.com/splunk/tarunner/internal/operator/transform"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/copy"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/noop"

	"github.com/splunk/tarunner/internal/conf"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
)

var CookFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"cook",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("When enabled, cook the data by applying props.conf"),
	featuregate.WithRegisterFromVersion("v0.1.0"),
)

func CreateOperatorConfigs(pCfg conf.Prop, transforms []conf.Transform) []operator.Config {
	var operators []operator.Config
	start := noop.NewConfigWithID(fmt.Sprintf("%s-start", pCfg.Name))
	switch pCfg.Type() {
	case conf.SourceType:
		start.IfExpr = fmt.Sprintf("attributes['com.splunk.sourcetype'] == %q", pCfg.Name)
	case conf.Default:
		// no condition
	case conf.Source:
		start.IfExpr = fmt.Sprintf("attributes['com.splunk.source'] == %q", pCfg.Name)
	case conf.Host:
		start.IfExpr = fmt.Sprintf("attributes['host.name'] == %q", pCfg.Name)
	default:
		panic(fmt.Sprintf("unknown prop type: %v", pCfg.Type()))
	}
	operators = append(operators, operator.NewConfig(start))
	var previous *helper.WriterConfig
	previous = &start.WriterConfig

	if pCfg.ShouldLineMerge {
		rec := recombine.NewConfigWithID(fmt.Sprintf("%s-recombine", pCfg.Name))
		previous.OutputIDs = []string{rec.OperatorID}
		operators = append(operators, operator.NewConfig(rec))
		previous = &rec.WriterConfig
	}

	if CookFeatureGate.IsEnabled() {
		for _, tCfg := range pCfg.Transforms {
			for _, stanza := range tCfg.Stanza {
				for _, tDef := range transforms {
					if tDef.Name == stanza {
						t := transform.NewConfig(pCfg.Name, tDef)
						previous.OutputIDs = []string{t.OperatorID}
						operators = append(operators, operator.NewConfig(t))
						previous = &t.WriterConfig
						break
					}
				}
			}
		}

		for _, fa := range pCfg.FieldAliases {
			copyOp := copy.NewConfigWithID(fmt.Sprintf("%s-copy", fa.Name))
			copyOp.From, _ = entry.NewField(fmt.Sprintf("attributes[%q]", fa.From))
			copyOp.To, _ = entry.NewField(fmt.Sprintf("attributes[%q]", fa.To))

			previous.OutputIDs = []string{copyOp.OperatorID}
			operators = append(operators, operator.NewConfig(copyOp))
			previous = &copyOp.WriterConfig
		}

		if pCfg.SourceType != "" {
			sourceTypeOp := copy.NewConfigWithID(fmt.Sprintf("%s-sourcetype", pCfg.Name))

			previous.OutputIDs = []string{sourceTypeOp.OperatorID}
			operators = append(operators, operator.NewConfig(sourceTypeOp))
			previous = &sourceTypeOp.WriterConfig
		}
	}

	endNoop := noop.NewConfigWithID(fmt.Sprintf("%s-end", pCfg.Name))
	previous.OutputIDs = []string{endNoop.OperatorID}
	endNoop.OutputIDs = []string{"end"}
	operators = append(operators, operator.NewConfig(endNoop))

	return operators
}
