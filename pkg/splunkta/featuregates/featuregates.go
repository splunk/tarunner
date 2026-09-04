// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package featuregates

import "go.opentelemetry.io/collector/featuregate"

var EnableDataCookingFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"enableDataCooking",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("When enabled, cook the data by applying index-time actions from props.conf and transforms.conf"),
	featuregate.WithRegisterFromVersion("v0.1.0"),
)
