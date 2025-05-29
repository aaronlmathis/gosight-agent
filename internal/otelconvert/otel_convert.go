/*
SPDX-License-Identifier: GPL-3.0-or-later

Copyright (C) 2025 Aaron Mathis aaron.mathis@gmail.com

This file is part of GoSight.

GoSight is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

GoSight is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with GoSight. If not, see https://www.gnu.org/licenses/.
*/

// Package otelconvert provides functions to convert GoSight data structures
// to OpenTelemetry Protocol (OTLP) format for transmission to OTLP-compatible
// endpoints and observability platforms.
package otelconvert

import (
	collogpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logpb "go.opentelemetry.io/proto/otlp/logs/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"

	"github.com/aaronlmathis/gosight-shared/model"
)

// ConvertToOTLPMetrics builds an OTLP ExportMetricsServiceRequest from a GoSight MetricPayload.
func ConvertToOTLPMetrics(payload *model.MetricPayload) *colmetricpb.ExportMetricsServiceRequest {
	if payload == nil || len(payload.Metrics) == 0 {
		return nil
	}

	resource := convertMetaToResource(payload.Meta)

	// Group metrics by namespace/subnamespace for proper scoping
	scopeMap := make(map[string][]*metricpb.Metric)

	for _, m := range payload.Metrics {
		scopeName := m.Namespace
		if m.SubNamespace != "" {
			scopeName = m.Namespace + "." + m.SubNamespace
		}

		var metric *metricpb.Metric

		// Handle different metric types based on whether StatisticValues is present
		if m.StatisticValues != nil && m.StatisticValues.SampleCount > 0 {
			// Convert to histogram if we have statistical data
			metric = &metricpb.Metric{
				Name: m.Name,
				Unit: m.Unit,
				Data: &metricpb.Metric_Histogram{
					Histogram: &metricpb.Histogram{
						AggregationTemporality: metricpb.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
						DataPoints: []*metricpb.HistogramDataPoint{
							{
								TimeUnixNano: uint64(m.Timestamp.UnixNano()),
								Attributes:   convertDimensions(m.Dimensions),
								Count:        uint64(m.StatisticValues.SampleCount),
								Sum:          &m.StatisticValues.Sum,
								Min:          &m.StatisticValues.Minimum,
								Max:          &m.StatisticValues.Maximum,
								// Note: You'd need bucket bounds/counts for full histogram
							},
						},
					},
				},
			}
		} else {
			// Convert to gauge for simple metrics
			metric = &metricpb.Metric{
				Name: m.Name,
				Unit: m.Unit,
				Data: &metricpb.Metric_Gauge{
					Gauge: &metricpb.Gauge{
						DataPoints: []*metricpb.NumberDataPoint{
							{
								TimeUnixNano: uint64(m.Timestamp.UnixNano()),
								Attributes:   convertDimensions(m.Dimensions),
								Value: &metricpb.NumberDataPoint_AsDouble{
									AsDouble: m.Value,
								},
							},
						},
					},
				},
			}
		}

		scopeMap[scopeName] = append(scopeMap[scopeName], metric)
	}

	// Create scope metrics for each namespace
	var scopeMetrics []*metricpb.ScopeMetrics
	for scopeName, metrics := range scopeMap {
		scopeMetrics = append(scopeMetrics, &metricpb.ScopeMetrics{
			Scope: &commonpb.InstrumentationScope{
				Name: scopeName,
			},
			Metrics: metrics,
		})
	}

	return &colmetricpb.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricpb.ResourceMetrics{
			{
				Resource:     resource,
				ScopeMetrics: scopeMetrics,
			},
		},
	}
}

// ConvertToOTLPLogs builds an OTLP ExportLogsServiceRequest from a GoSight LogPayload.
// This function ensures that host_id and agent_id are preserved in the resource attributes
// to maintain proper identification and correlation of log data in OTLP-compatible systems.
func ConvertToOTLPLogs(payload *model.LogPayload) *collogpb.ExportLogsServiceRequest {
	if payload == nil || len(payload.Logs) == 0 {
		return nil
	}

	// Convert Meta to Resource, ensuring host_id and agent_id are included
	resource := convertLogPayloadToResource(payload)

	// Group logs by source for proper scoping
	scopeMap := make(map[string][]*logpb.LogRecord)

	for _, logEntry := range payload.Logs {
		scopeName := logEntry.Source
		if scopeName == "" {
			scopeName = "unknown"
		}

		// Convert log level to OTLP severity
		severityNumber := convertLogLevelToSeverity(logEntry.Level)

		// Create log record
		logRecord := &logpb.LogRecord{
			TimeUnixNano:   uint64(logEntry.Timestamp.UnixNano()),
			SeverityNumber: severityNumber,
			SeverityText:   logEntry.Level,
			Body:           &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: logEntry.Message}},
			Attributes:     convertLogAttributes(logEntry),
		}

		scopeMap[scopeName] = append(scopeMap[scopeName], logRecord)
	}

	// Create scope logs for each source
	var scopeLogs []*logpb.ScopeLogs
	for scopeName, logRecords := range scopeMap {
		scopeLogs = append(scopeLogs, &logpb.ScopeLogs{
			Scope: &commonpb.InstrumentationScope{
				Name: scopeName,
			},
			LogRecords: logRecords,
		})
	}

	return &collogpb.ExportLogsServiceRequest{
		ResourceLogs: []*logpb.ResourceLogs{
			{
				Resource:  resource,
				ScopeLogs: scopeLogs,
			},
		},
	}
}

// convertDimensions converts a map of string dimensions to OTLP KeyValue attributes.
func convertDimensions(dims map[string]string) []*commonpb.KeyValue {
	out := make([]*commonpb.KeyValue, 0, len(dims))
	for k, v := range dims {
		if k != "" && v != "" {
			out = append(out, &commonpb.KeyValue{
				Key:   k,
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: v}},
			})
		}
	}
	return out
}

// convertMetaToResource converts GoSight Meta information to OTLP Resource attributes.
func convertMetaToResource(meta *model.Meta) *resourcepb.Resource {
	if meta == nil {
		return &resourcepb.Resource{}
	}

	attrs := []*commonpb.KeyValue{}

	add := func(key, val string) {
		if val != "" {
			attrs = append(attrs, &commonpb.KeyValue{
				Key:   key,
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: val}},
			})
		}
	}

	// Core identity
	add("host.id", meta.HostID)
	add("host.name", meta.Hostname)
	add("agent.id", meta.AgentID)
	add("resource.id", meta.ResourceID)
	add("resource.kind", meta.Kind)
	add("agent.version", meta.AgentVersion)
	add("endpoint.id", meta.EndpointID)

	// OS / Platform
	add("os.type", meta.OS)
	add("os.version", meta.OSVersion)
	add("platform", meta.Platform)
	add("platform.family", meta.PlatformFamily)
	add("platform.version", meta.PlatformVersion)
	add("arch", meta.Architecture)
	add("kernel.version", meta.KernelVersion)
	add("kernel.architecture", meta.KernelArchitecture)

	// Cloud
	add("cloud.provider", meta.CloudProvider)
	add("cloud.region", meta.Region)
	add("cloud.zone", meta.AvailabilityZone)
	add("cloud.account.id", meta.AccountID)
	add("cloud.project.id", meta.ProjectID)
	add("cloud.instance.id", meta.InstanceID)
	add("cloud.instance.type", meta.InstanceType)
	add("cloud.resource.group", meta.ResourceGroup)
	add("cloud.vpc.id", meta.VPCID)
	add("cloud.subnet.id", meta.SubnetID)
	add("cloud.image.id", meta.ImageID)
	add("cloud.service.id", meta.ServiceID)

	// Container / Kubernetes
	add("container.id", meta.ContainerID)
	add("container.name", meta.ContainerName)
	add("container.image.id", meta.ContainerImageID)
	add("container.image.name", meta.ContainerImageName)
	add("k8s.pod.name", meta.PodName)
	add("k8s.namespace.name", meta.Namespace)
	add("k8s.cluster.name", meta.ClusterName)
	add("k8s.node.name", meta.NodeName)

	// App
	add("application", meta.Application)
	add("service.name", meta.Service)
	add("service.version", meta.Version)
	add("environment", meta.Environment)
	add("deployment.id", meta.DeploymentID)

	// Network
	add("host.ip", meta.IPAddress)
	add("host.public_ip", meta.PublicIP)
	add("host.private_ip", meta.PrivateIP)
	add("host.mac", meta.MACAddress)
	add("network.interface", meta.NetworkInterface)

	// Labels
	for k, v := range meta.Labels {
		add("tag."+k, v)
	}

	return &resourcepb.Resource{Attributes: attrs}
}

// convertLogPayloadToResource creates an OTLP Resource from LogPayload, ensuring host_id and agent_id are preserved
func convertLogPayloadToResource(payload *model.LogPayload) *resourcepb.Resource {
	attrs := []*commonpb.KeyValue{}

	add := func(key, val string) {
		if val != "" {
			attrs = append(attrs, &commonpb.KeyValue{
				Key:   key,
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: val}},
			})
		}
	}

	// Core identity from LogPayload - CRUCIAL for proper identification
	add("host.id", payload.HostID)
	add("agent.id", payload.AgentID)
	add("host.name", payload.Hostname)
	add("endpoint.id", payload.EndpointID)

	// If Meta is available, use the detailed metadata conversion
	if payload.Meta != nil {
		metaResource := convertMetaToResource(payload.Meta)
		// Merge meta attributes, but preserve the core identity fields from LogPayload
		for _, attr := range metaResource.Attributes {
			// Skip if we already added these core fields from LogPayload
			if attr.Key != "host.id" && attr.Key != "agent.id" &&
				attr.Key != "host.name" && attr.Key != "endpoint.id" {
				attrs = append(attrs, attr)
			}
		}
	}

	return &resourcepb.Resource{Attributes: attrs}
}

// convertLogAttributes converts log entry fields, tags, and metadata to OTLP attributes
func convertLogAttributes(logEntry model.LogEntry) []*commonpb.KeyValue {
	attrs := []*commonpb.KeyValue{}

	add := func(key, val string) {
		if val != "" {
			attrs = append(attrs, &commonpb.KeyValue{
				Key:   key,
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: val}},
			})
		}
	}

	addInt := func(key string, val int) {
		if val > 0 {
			attrs = append(attrs, &commonpb.KeyValue{
				Key:   key,
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: int64(val)}},
			})
		}
	}

	// Basic log attributes
	add("log.source", logEntry.Source)
	add("log.category", logEntry.Category)
	addInt("process.pid", logEntry.PID)

	// Add structured fields
	for k, v := range logEntry.Fields {
		add("field."+k, v)
	}

	// Add tags
	for k, v := range logEntry.Labels {
		add("tag."+k, v)
	}

	// Add log metadata if present
	if logEntry.Meta != nil {
		add("log.platform", logEntry.Meta.Platform)
		add("log.app_name", logEntry.Meta.AppName)
		add("log.app_version", logEntry.Meta.AppVersion)
		add("log.container_id", logEntry.Meta.ContainerID)
		add("log.container_name", logEntry.Meta.ContainerName)
		add("log.unit", logEntry.Meta.Unit)
		add("log.service", logEntry.Meta.Service)
		add("log.event_id", logEntry.Meta.EventID)
		add("log.user", logEntry.Meta.User)
		add("log.executable", logEntry.Meta.Executable)
		add("log.path", logEntry.Meta.Path)

		// Add extra metadata
		for k, v := range logEntry.Meta.Extra {
			add("log.extra."+k, v)
		}
	}

	return attrs
}

// convertLogLevelToSeverity converts string log levels to OTLP severity numbers
func convertLogLevelToSeverity(level string) logpb.SeverityNumber {
	switch level {
	case "trace", "TRACE":
		return logpb.SeverityNumber_SEVERITY_NUMBER_TRACE
	case "debug", "DEBUG":
		return logpb.SeverityNumber_SEVERITY_NUMBER_DEBUG
	case "info", "INFO":
		return logpb.SeverityNumber_SEVERITY_NUMBER_INFO
	case "warn", "WARN", "warning", "WARNING":
		return logpb.SeverityNumber_SEVERITY_NUMBER_WARN
	case "error", "ERROR":
		return logpb.SeverityNumber_SEVERITY_NUMBER_ERROR
	case "fatal", "FATAL", "critical", "CRITICAL":
		return logpb.SeverityNumber_SEVERITY_NUMBER_FATAL
	default:
		return logpb.SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED
	}
}
