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
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/aaronlmathis/gosight-shared/model"
	collogpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logpb "go.opentelemetry.io/proto/otlp/logs/v1"
)

// OTLPToLogEntries converts an OTLP ExportLogsServiceRequest into GoSight’s []*model.LogEntry.
func OTLPToLogEntries(req *collogpb.ExportLogsServiceRequest) []*model.LogEntry {
	var entries []*model.LogEntry

	for _, resourceLogs := range req.ResourceLogs {
		// Convert resource-level attributes (e.g., service.name, k8s.pod.name, etc.)
		resourceAttrs := convertKeyValueToStringMap(resourceLogs.Resource.Attributes)
		// Build a Meta object from those resource attributes
		meta := buildMetaFromResourceAttrs(resourceAttrs)

		for _, scopeLogs := range resourceLogs.ScopeLogs {
			// You could also record scopeLogs.Scope.Name / Version if desired
			for _, lr := range scopeLogs.LogRecords {
				// Event timestamp
				timestamp := time.Unix(0, int64(lr.TimeUnixNano))
				// Observed timestamp (when collector saw it)
				observed := time.Unix(0, int64(lr.ObservedTimeUnixNano))
				// Convert record-level attributes into a map[string]interface{}
				attrs := convertAnyValueMap(lr.Attributes)

				// Build the LogEntry
				entry := &model.LogEntry{
					Timestamp:         timestamp,
					ObservedTimestamp: observed,
					SeverityText:      lr.SeverityText,
					SeverityNumber:    int32(lr.SeverityNumber),

					Body:  lr.Body.GetStringValue(),
					Flags: lr.Flags,

					Level:    lr.SeverityText,
					Message:  lr.Body.GetStringValue(),
					Source:   resourceAttrs["service.name"], // e.g., service name if set
					Category: "",                            // populate if you have a “category” attribute

					Fields:     nil, // populate if you parse JSON‐style fields inside Attributes
					Labels:     nil, // populate if you have any labels to attach separately
					Attributes: attrs,
					Meta:       meta,
				}

				if len(lr.TraceId) == 16 {
					entry.TraceID = hex.EncodeToString(lr.TraceId)
				}
				if len(lr.SpanId) == 8 {
					entry.SpanID = hex.EncodeToString(lr.SpanId)
				}

				// If a “pid” attribute exists and is numeric, set entry.PID
				if pidStr, ok := attrs["pid"].(string); ok {
					if pid, err := strconv.Atoi(pidStr); err == nil {
						entry.PID = pid
					}
				}

				entries = append(entries, entry)
			}
		}
	}

	return entries
}

// ConvertToOTLPLogs builds an OTLP ExportLogsServiceRequest from a slice of GoSight LogEntries.
func ConvertToOTLPLogs(logs []model.LogEntry) *collogpb.ExportLogsServiceRequest {
	if len(logs) == 0 {
		return nil
	}

	// Group logs by resource (Meta) and scope (source)
	resourceMap := make(map[*model.Meta]map[string][]model.LogEntry)

	for _, logEntry := range logs {
		if resourceMap[logEntry.Meta] == nil {
			resourceMap[logEntry.Meta] = make(map[string][]model.LogEntry)
		}

		scopeName := logEntry.Source
		if scopeName == "" {
			scopeName = "unknown"
		}

		resourceMap[logEntry.Meta][scopeName] = append(resourceMap[logEntry.Meta][scopeName], logEntry)
	}

	var resourceLogs []*logpb.ResourceLogs

	for meta, scopeMap := range resourceMap {
		resource := convertMetaToResource(meta)
		var scopeLogs []*logpb.ScopeLogs

		for scopeName, logsInScope := range scopeMap {
			var logRecords []*logpb.LogRecord

			for _, logEntry := range logsInScope {
				// Convert log level to OTLP severity
				severityNumber := convertLogLevelToSeverity(logEntry.Level)

				// Create log record
				logRecord := &logpb.LogRecord{
					TimeUnixNano:         uint64(logEntry.Timestamp.UnixNano()),
					ObservedTimeUnixNano: uint64(logEntry.ObservedTimestamp.UnixNano()),
					SeverityNumber:       severityNumber,
					SeverityText:         logEntry.Level,
					Body:                 &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: logEntry.Message}},
					Attributes:           convertLogAttributes(logEntry),
				}

				// Add trace context if available
				if logEntry.TraceID != "" {
					if traceBytes, err := hex.DecodeString(logEntry.TraceID); err == nil && len(traceBytes) == 16 {
						logRecord.TraceId = traceBytes
					}
				}
				if logEntry.SpanID != "" {
					if spanBytes, err := hex.DecodeString(logEntry.SpanID); err == nil && len(spanBytes) == 8 {
						logRecord.SpanId = spanBytes
					}
				}
				if logEntry.Flags > 0 {
					logRecord.Flags = logEntry.Flags
				}

				logRecords = append(logRecords, logRecord)
			}

			scopeLogs = append(scopeLogs, &logpb.ScopeLogs{
				Scope: &commonpb.InstrumentationScope{
					Name: scopeName,
				},
				LogRecords: logRecords,
			})
		}

		resourceLogs = append(resourceLogs, &logpb.ResourceLogs{
			Resource:  resource,
			ScopeLogs: scopeLogs,
		})
	}

	return &collogpb.ExportLogsServiceRequest{
		ResourceLogs: resourceLogs,
	}
}

// decodeHexOrNil decodes a hex string into a byte slice of the given length, or returns nil if invalid.
func decodeHexOrNil(hexStr string, wantLen int) []byte {
	if len(hexStr) != wantLen*2 {
		return nil
	}
	b := make([]byte, wantLen)
	_, err := fmt.Sscanf(hexStr, "%x", &b)
	if err != nil {
		return nil
	}
	return b
}

// convertLogAttributes converts log entry fields, labels, and metadata to OTLP attributes
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

	// Add labels
	for k, v := range logEntry.Labels {
		add("label."+k, v)
	}

	// Add attributes directly
	for k, v := range logEntry.Attributes {
		switch val := v.(type) {
		case string:
			add("attr."+k, val)
		case int:
			addInt("attr."+k, val)
		case int64:
			attrs = append(attrs, &commonpb.KeyValue{
				Key:   "attr." + k,
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: val}},
			})
		case float64:
			attrs = append(attrs, &commonpb.KeyValue{
				Key:   "attr." + k,
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: val}},
			})
		case bool:
			attrs = append(attrs, &commonpb.KeyValue{
				Key:   "attr." + k,
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_BoolValue{BoolValue: val}},
			})
		}
	}

	// Add metadata from Meta if present
	if logEntry.Meta != nil {
		add("host.id", logEntry.Meta.HostID)
		add("agent.id", logEntry.Meta.AgentID)
		add("host.name", logEntry.Meta.Hostname)
		add("endpoint.id", logEntry.Meta.EndpointID)
		add("container.id", logEntry.Meta.ContainerID)
		add("container.name", logEntry.Meta.ContainerName)
		add("service.name", logEntry.Meta.Service)
		add("service.version", logEntry.Meta.AppVersion)

		// Add custom tags and labels
		for k, v := range logEntry.Meta.Tags {
			add("tag."+k, v)
		}
		for k, v := range logEntry.Meta.Labels {
			add("meta.label."+k, v)
		}
	}

	return attrs
}
