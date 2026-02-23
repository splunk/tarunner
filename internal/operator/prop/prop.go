// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prop

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"

	"github.com/splunk/tarunner/internal/conf"
)

// Parser is an operator applying a prop config to an entry.
type Parser struct {
	helper.ParserOperator
	config conf.Prop
}

func (p *Parser) ProcessBatch(ctx context.Context, entries []*entry.Entry) error {
	return p.ProcessBatchWith(ctx, entries, p.parse)
}

// Process will parse an entry for regex.
func (p *Parser) Process(ctx context.Context, entry *entry.Entry) error {
	return p.ProcessWith(ctx, entry, p.parse)
}

func (p *Parser) parse(value any) (any, error) {

}
