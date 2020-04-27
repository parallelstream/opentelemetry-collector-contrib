// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collection

import (
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8sclusterreceiver/utils"
)

const (
	// Keys for container properties.
	containerKeyStatus       = "container.status"
	containerKeyStatusReason = "container.status.reason"

	// Values for container properties
	containerStatusRunning    = "running"
	containerStatusWaiting    = "waiting"
	containerStatusTerminated = "terminated"
)

var containerRestartMetric = &metricspb.MetricDescriptor{
	Name: "kubernetes/container/restarts",
	Description: "How many times the container has restarted in the recent past. " +
		"This value is pulled directly from the K8s API and the value can go indefinitely high" +
		" and be reset to 0 at any time depending on how your kubelet is configured to prune" +
		" dead containers. It is best to not depend too much on the exact value but rather look" +
		" at it as either == 0, in which case you can conclude there were no restarts in the recent" +
		" past, or > 0, in which case you can conclude there were restarts in the recent past, and" +
		" not try and analyze the value beyond that.",
	Unit: "1",
	Type: metricspb.MetricDescriptor_GAUGE_INT64,
}

var containerReadyMetric = &metricspb.MetricDescriptor{
	Name:        "kubernetes/container/ready",
	Description: "Whether a container has passed its readiness probe (0 for no, 1 for yes)",
	Type:        metricspb.MetricDescriptor_GAUGE_INT64,
}

// getStatusMetricsForContainer returns metrics about the status of the container.
func getStatusMetricsForContainer(cs corev1.ContainerStatus) []*metricspb.Metric {
	metrics := []*metricspb.Metric{
		{
			MetricDescriptor: containerRestartMetric,
			Timeseries: []*metricspb.TimeSeries{
				utils.GetInt64TimeSeries(int64(cs.RestartCount)),
			},
		},
		{
			MetricDescriptor: containerReadyMetric,
			Timeseries: []*metricspb.TimeSeries{
				utils.GetInt64TimeSeries(boolToInt64(cs.Ready)),
			},
		},
	}

	return metrics
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

var containerRequestMetric = &metricspb.MetricDescriptor{
	Name:        "kubernetes/container/request",
	Description: "Resource requested for the container",
	Type:        metricspb.MetricDescriptor_GAUGE_INT64,
	LabelKeys:   []*metricspb.LabelKey{{Key: "resource"}},
}

var containerLimitMetric = &metricspb.MetricDescriptor{
	Name:        "kubernetes/container/limit",
	Description: "Maximum resource limit set for the container",
	Type:        metricspb.MetricDescriptor_GAUGE_INT64,
	LabelKeys:   []*metricspb.LabelKey{{Key: "resource"}},
}

// getSpecMetricsForContainer metricizes values from the container spec.
// This includes values like resource requests and limits.
func getSpecMetricsForContainer(c corev1.Container) []*metricspb.Metric {
	metrics := make([]*metricspb.Metric, 0)

	for _, t := range []struct {
		metric *metricspb.MetricDescriptor
		rl     corev1.ResourceList
	}{
		{
			containerRequestMetric,
			c.Resources.Requests,
		},
		{
			containerLimitMetric,
			c.Resources.Limits,
		},
	} {
		for k, v := range t.rl {
			val := v.Value()
			if k == corev1.ResourceCPU {
				val = v.MilliValue()
			}

			metrics = append(metrics,
				&metricspb.Metric{
					MetricDescriptor: t.metric,
					Timeseries: []*metricspb.TimeSeries{
						utils.GetInt64TimeSeriesWithLabels(val, []*metricspb.LabelValue{{Value: string(k)}}),
					},
				},
			)
		}
	}

	return metrics
}

// getResourceForContainer returns a proto representation of the pod.
func getResourceForContainer(labels map[string]string) *resourcepb.Resource {
	return &resourcepb.Resource{
		Type:   containerType,
		Labels: labels,
	}
}

// getAllContainerLabels returns all container labels, including ones from
// the pod in which the container is running.
func getAllContainerLabels(cs corev1.ContainerStatus,
	dims map[string]string) map[string]string {

	out := utils.CloneStringMap(dims)

	out[containerKeyID] = utils.StripContainerID(cs.ContainerID)
	out[containerKeySpecName] = cs.Name
	out[conventions.AttributeContainerImage] = cs.Image

	return out
}

func getMetadataForContainer(cs corev1.ContainerStatus) *KubernetesMetadata {
	properties := map[string]string{}

	if cs.State.Running != nil {
		properties[containerKeyStatus] = containerStatusRunning
	}

	if cs.State.Terminated != nil {
		properties[containerKeyStatus] = containerStatusTerminated
		properties[containerKeyStatusReason] = cs.State.Terminated.Reason
	}

	if cs.State.Waiting != nil {
		properties[containerKeyStatus] = containerStatusWaiting
		properties[containerKeyStatusReason] = cs.State.Waiting.Reason
	}

	return &KubernetesMetadata{
		resourceIDKey: containerKeyID,
		resourceID:    utils.StripContainerID(cs.ContainerID),
		properties:    properties,
	}
}