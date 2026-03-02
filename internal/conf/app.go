// Copyright Splunk, Inc.
// SPDX-License-Identifier: Apache-2.0

package conf

type App struct {
	Dir        string
	Name       string
	Inputs     []Input
	Transforms []Transform
	Props      []Prop
}
