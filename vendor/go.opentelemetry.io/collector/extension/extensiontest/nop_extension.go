// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensiontest // import "go.opentelemetry.io/collector/extension/extensiontest"

import (
	"context"

	"github.com/google/uuid"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension"
)

var nopType = component.MustNewType("nop")

// NewNopSettings returns a new nop settings for extension.Factory Create* functions.
func NewNopSettings() extension.Settings {
	return extension.Settings{
		ID:                component.NewIDWithName(nopType, uuid.NewString()),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// NewNopFactory returns an extension.Factory that constructs nop extensions.
func NewNopFactory() extension.Factory {
	return extension.NewFactory(
		nopType,
		func() component.Config {
			return &nopConfig{}
		},
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return nopInstance, nil
		},
		component.StabilityLevelStable)
}

type nopConfig struct{}

var nopInstance = &nopExtension{}

// nopExtension acts as an extension for testing purposes.
type nopExtension struct {
	component.StartFunc
	component.ShutdownFunc
}