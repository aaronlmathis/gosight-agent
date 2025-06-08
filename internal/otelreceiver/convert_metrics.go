// SPDX-License-Identifier: GPL-3.0-or-later

// Copyright (C) 2025 Aaron Mathis <aaron.mathis@gmail.com>

// This file is part of GoSight.

// GoSight is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// GoSight is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with GoSight. If not, see https://www.gnu.org/licenses/.
//

package otelreceiver

import (
	"time"

	"github.com/aaronlmathis/gosight-shared/model"
	otlpcolpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

// OTLPToMetrics converts an OTLP ExportMetricsServiceRequest into GoSight’s []*model.Metric.
func OTLPToMetrics(req *otlpcolpb.ExportMetricsServiceRequest) []*model.Metric {
	var out []*model.Metric

	for _, rm := range req.ResourceMetrics {
		resourceAttrs := convertKeyValueToStringMap(rm.Resource.Attributes)
		meta := buildMetaFromResourceAttrs(resourceAttrs)
		for _, ilm := range rm.ScopeMetrics {
			for _, om := range ilm.Metrics {
				m := &model.Metric{
					Name:                   om.Name,
					Description:            om.Description,
					Unit:                   om.Unit,
					DataType:               "",
					AggregationTemporality: "",
					DataPoints:             nil,
					StorageResolution:      1,
					Source:                 "otlp",
					Meta:                   meta,
				}

				switch data := om.Data.(type) {

				case *metricspb.Metric_Gauge:
					m.DataType = "gauge"
					for _, od := range data.Gauge.DataPoints {
						value := extractNumberDataPointValue(od)
						dp := model.DataPoint{
							Attributes: convertKeyValueToMap(od.Attributes),
							Timestamp:  time.Unix(0, int64(od.TimeUnixNano)),
							Value:      value,
							Exemplars:  convertOtelExemplars(od.Exemplars),
						}
						m.DataPoints = append(m.DataPoints, dp)
					}

				case *metricspb.Metric_Sum:
					m.DataType = "sum"
					m.AggregationTemporality = data.Sum.AggregationTemporality.String()
					for _, od := range data.Sum.DataPoints {
						value := extractNumberDataPointValue(od)
						dp := model.DataPoint{
							Attributes:     convertKeyValueToMap(od.Attributes),
							StartTimestamp: time.Unix(0, int64(od.StartTimeUnixNano)),
							Timestamp:      time.Unix(0, int64(od.TimeUnixNano)),
							Value:          value,
							Exemplars:      convertOtelExemplars(od.Exemplars),
						}
						m.DataPoints = append(m.DataPoints, dp)
					}

				case *metricspb.Metric_Histogram:
					m.DataType = "histogram"
					m.AggregationTemporality = data.Histogram.AggregationTemporality.String()
					for _, od := range data.Histogram.DataPoints {
						dp := model.DataPoint{
							Attributes:     convertKeyValueToMap(od.Attributes),
							StartTimestamp: time.Unix(0, int64(od.StartTimeUnixNano)),
							Timestamp:      time.Unix(0, int64(od.TimeUnixNano)),
							Count:          od.GetCount(),
							Sum:            od.GetSum(),
							BucketCounts:   od.BucketCounts,
							ExplicitBounds: od.ExplicitBounds,
							Exemplars:      convertOtelExemplars(od.Exemplars),
						}
						m.DataPoints = append(m.DataPoints, dp)
					}

				case *metricspb.Metric_Summary:
					m.DataType = "summary"
					for _, od := range data.Summary.DataPoints {
						var qvs []model.QuantileValue
						for _, qt := range od.QuantileValues {
							qvs = append(qvs, model.QuantileValue{
								Quantile: qt.GetQuantile(),
								Value:    qt.GetValue(),
							})
						}
						dp := model.DataPoint{
							Attributes:     convertKeyValueToMap(od.Attributes),
							StartTimestamp: time.Unix(0, int64(od.StartTimeUnixNano)),
							Timestamp:      time.Unix(0, int64(od.TimeUnixNano)),
							Count:          od.GetCount(),
							Sum:            od.GetSum(),
							QuantileValues: qvs,
						}
						m.DataPoints = append(m.DataPoints, dp)
					}

				default:
					// Unknown or unsupported metric type—skip it.
					continue
				}

				out = append(out, m)
			}
		}
	}

	return out
}

// ConvertToOTLPMetrics builds an OTLP ExportMetricsServiceRequest from a slice of GoSight Metrics.
func ConvertToOTLPMetrics(metrics []*model.Metric) *otlpcolpb.ExportMetricsServiceRequest {

	if len(metrics) == 0 {
		return nil
	}

	// Group metrics by resource (Meta) and scope (namespace)
	resourceMap := make(map[*model.Meta]map[string][]*model.Metric)

	for _, metric := range metrics {
		if metric == nil {
			continue
		}

		if resourceMap[metric.Meta] == nil {
			resourceMap[metric.Meta] = make(map[string][]*model.Metric)
		}

		scopeName := metric.Namespace
		if metric.SubNamespace != "" {
			scopeName = metric.Namespace + "." + metric.SubNamespace
		}

		resourceMap[metric.Meta][scopeName] = append(resourceMap[metric.Meta][scopeName], metric)
	}

	var resourceMetrics []*metricspb.ResourceMetrics

	for meta, scopeMap := range resourceMap {
		resource := convertMetaToResource(meta)
		var scopeMetrics []*metricspb.ScopeMetrics

		for scopeName, metricsInScope := range scopeMap {
			var otlpMetrics []*metricspb.Metric

			for _, metric := range metricsInScope {
				otlpMetric := &metricspb.Metric{
					Name:        metric.Name,
					Description: metric.Description,
					Unit:        metric.Unit,
				}

				switch metric.DataType {
				case "gauge":
					var dataPoints []*metricspb.NumberDataPoint
					for _, dp := range metric.DataPoints {
						ndp := &metricspb.NumberDataPoint{
							TimeUnixNano: uint64(dp.Timestamp.UnixNano()),
							Attributes:   convertStringMapToKeyValue(dp.Attributes),
							Value:        &metricspb.NumberDataPoint_AsDouble{AsDouble: dp.Value},
						}
						dataPoints = append(dataPoints, ndp)
					}
					otlpMetric.Data = &metricspb.Metric_Gauge{
						Gauge: &metricspb.Gauge{
							DataPoints: dataPoints,
						},
					}
				case "sum":
					var dataPoints []*metricspb.NumberDataPoint
					for _, dp := range metric.DataPoints {
						ndp := &metricspb.NumberDataPoint{
							StartTimeUnixNano: uint64(dp.StartTimestamp.UnixNano()),
							TimeUnixNano:      uint64(dp.Timestamp.UnixNano()),
							Attributes:        convertStringMapToKeyValue(dp.Attributes),
							Value:             &metricspb.NumberDataPoint_AsDouble{AsDouble: dp.Value},
						}
						dataPoints = append(dataPoints, ndp)
					}
					temporality := metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_UNSPECIFIED
					if metric.AggregationTemporality == "delta" || metric.AggregationTemporality == "AGGREGATION_TEMPORALITY_DELTA" {
						temporality = metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA
					} else if metric.AggregationTemporality == "cumulative" || metric.AggregationTemporality == "AGGREGATION_TEMPORALITY_CUMULATIVE" {
						temporality = metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE
					}
					otlpMetric.Data = &metricspb.Metric_Sum{
						Sum: &metricspb.Sum{
							DataPoints:             dataPoints,
							AggregationTemporality: temporality,
						},
					}
				case "histogram":
					var dataPoints []*metricspb.HistogramDataPoint
					for _, dp := range metric.DataPoints {
						sum := dp.Sum
						hdp := &metricspb.HistogramDataPoint{
							StartTimeUnixNano: uint64(dp.StartTimestamp.UnixNano()),
							TimeUnixNano:      uint64(dp.Timestamp.UnixNano()),
							Count:             dp.Count,
							Sum:               &sum,
							BucketCounts:      dp.BucketCounts,
							ExplicitBounds:    dp.ExplicitBounds,
							Attributes:        convertStringMapToKeyValue(dp.Attributes),
						}
						dataPoints = append(dataPoints, hdp)
					}
					temporality := metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_UNSPECIFIED
					if metric.AggregationTemporality == "delta" || metric.AggregationTemporality == "AGGREGATION_TEMPORALITY_DELTA" {
						temporality = metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA
					} else if metric.AggregationTemporality == "cumulative" || metric.AggregationTemporality == "AGGREGATION_TEMPORALITY_CUMULATIVE" {
						temporality = metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE
					}
					otlpMetric.Data = &metricspb.Metric_Histogram{
						Histogram: &metricspb.Histogram{
							DataPoints:             dataPoints,
							AggregationTemporality: temporality,
						},
					}
				case "summary":
					var dataPoints []*metricspb.SummaryDataPoint
					for _, dp := range metric.DataPoints {
						var qvs []*metricspb.SummaryDataPoint_ValueAtQuantile
						for _, qv := range dp.QuantileValues {
							qvs = append(qvs, &metricspb.SummaryDataPoint_ValueAtQuantile{
								Quantile: qv.Quantile,
								Value:    qv.Value,
							})
						}
						sdp := &metricspb.SummaryDataPoint{
							StartTimeUnixNano: uint64(dp.StartTimestamp.UnixNano()),
							TimeUnixNano:      uint64(dp.Timestamp.UnixNano()),
							Count:             dp.Count,
							Sum:               dp.Sum,
							QuantileValues:    qvs,
							Attributes:        convertStringMapToKeyValue(dp.Attributes),
						}
						dataPoints = append(dataPoints, sdp)
					}
					otlpMetric.Data = &metricspb.Metric_Summary{
						Summary: &metricspb.Summary{
							DataPoints: dataPoints,
						},
					}
				}

				otlpMetrics = append(otlpMetrics, otlpMetric)
			}

			scopeMetrics = append(scopeMetrics, &metricspb.ScopeMetrics{
				Scope: &commonpb.InstrumentationScope{
					Name: scopeName,
				},
				Metrics: otlpMetrics,
			})
		}

		resourceMetrics = append(resourceMetrics, &metricspb.ResourceMetrics{
			Resource:     resource,
			ScopeMetrics: scopeMetrics,
		})
	}

	return &otlpcolpb.ExportMetricsServiceRequest{
		ResourceMetrics: resourceMetrics,
	}
}

// convertStringMapToKeyValue converts a map[string]string to OTLP KeyValue attributes.
func convertStringMapToKeyValue(m map[string]string) []*commonpb.KeyValue {
	out := make([]*commonpb.KeyValue, 0, len(m))
	for k, v := range m {
		if k != "" && v != "" {
			out = append(out, &commonpb.KeyValue{
				Key:   k,
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: v}},
			})
		}
	}
	return out
}
