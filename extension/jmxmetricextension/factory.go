// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jmxmetricextension

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/extension/extensionhelper"
)

const (
	typeStr            = "jmx_metrics"
	otlpExporter       = "otlp"
	otlpEndpoint       = "localhost:55680"
	prometheusExporter = "prometheus"
	prometheusEndpoint = "localhost"
	prometheusPort     = 9090
)

func NewFactory() component.ExtensionFactory {
	return extensionhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		createExtension)
}

func createDefaultConfig() configmodels.Extension {
	return &config{
		ExtensionSettings: configmodels.ExtensionSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		JARPath:        "/opt/opentelemetry-java-contrib-jmx-metrics.jar",
		Interval:       10 * time.Second,
		Exporter:       otlpExporter,
		OTLPEndpoint:   otlpEndpoint,
		OTLPTimeout:    5 * time.Second,
		PrometheusHost: prometheusEndpoint,
		PrometheusPort: prometheusPort,
	}
}

func createExtension(
	ctx context.Context,
	params component.ExtensionCreateParams,
	cfg configmodels.Extension,
) (component.ServiceExtension, error) {
	jmxConfig := cfg.(*config)
	if err := jmxConfig.validate(); err != nil {
		return nil, err
	}
	return newJMXMetricExtension(params.Logger, jmxConfig), nil
}
