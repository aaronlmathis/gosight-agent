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

package otelconvert

import (
	"testing"
	"time"

	"github.com/aaronlmathis/gosight-shared/model"
)

func TestConvertToOTLPLogs(t *testing.T) {
	// Test data
	testTime := time.Now()
	logPayload := &model.LogPayload{
		AgentID:    "test-agent-123",
		HostID:     "test-host-456",
		Hostname:   "test-hostname",
		EndpointID: "test-endpoint-789",
		Timestamp:  testTime,
		Logs: []model.LogEntry{
			{
				Timestamp: testTime,
				Level:     "info",
				Message:   "Test log message",
				Source:    "test-source",
				Category:  "test",
				PID:       1234,
				Fields: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				Tags: map[string]string{
					"env": "test",
					"app": "gosight",
				},
				Meta: &model.LogMeta{
					Platform:    "linux",
					AppName:     "test-app",
					AppVersion:  "1.0.0",
					ContainerID: "container-123",
					Unit:        "test.service",
				},
			},
		},
		Meta: &model.Meta{
			AgentID:    "test-agent-123",
			HostID:     "test-host-456",
			Hostname:   "test-hostname",
			EndpointID: "test-endpoint-789",
			Kind:       "host",
			OS:         "linux",
			Platform:   "ubuntu",
		},
	}

	// Convert to OTLP
	otlpRequest := ConvertToOTLPLogs(logPayload)

	// Basic checks
	if otlpRequest == nil {
		t.Fatal("ConvertToOTLPLogs returned nil")
	}

	if len(otlpRequest.ResourceLogs) != 1 {
		t.Fatalf("Expected 1 ResourceLogs, got %d", len(otlpRequest.ResourceLogs))
	}

	resourceLog := otlpRequest.ResourceLogs[0]
	if resourceLog.Resource == nil {
		t.Fatal("Resource is nil")
	}

	// Check that host_id and agent_id are preserved in resource attributes
	var foundHostID, foundAgentID bool
	for _, attr := range resourceLog.Resource.Attributes {
		switch attr.Key {
		case "host.id":
			if attr.Value.GetStringValue() == "test-host-456" {
				foundHostID = true
			}
		case "agent.id":
			if attr.Value.GetStringValue() == "test-agent-123" {
				foundAgentID = true
			}
		}
	}

	if !foundHostID {
		t.Error("host.id not found in resource attributes")
	}
	if !foundAgentID {
		t.Error("agent.id not found in resource attributes")
	}

	// Check log records
	if len(resourceLog.ScopeLogs) != 1 {
		t.Fatalf("Expected 1 ScopeLogs, got %d", len(resourceLog.ScopeLogs))
	}

	scopeLog := resourceLog.ScopeLogs[0]
	if len(scopeLog.LogRecords) != 1 {
		t.Fatalf("Expected 1 LogRecord, got %d", len(scopeLog.LogRecords))
	}

	logRecord := scopeLog.LogRecords[0]
	if logRecord.SeverityText != "info" {
		t.Errorf("Expected severity 'info', got '%s'", logRecord.SeverityText)
	}

	if logRecord.Body.GetStringValue() != "Test log message" {
		t.Errorf("Expected message 'Test log message', got '%s'", logRecord.Body.GetStringValue())
	}
}

func TestConvertToOTLPMetrics(t *testing.T) {
	// Test data
	testTime := time.Now()
	metricPayload := &model.MetricPayload{
		AgentID:    "test-agent-123",
		HostID:     "test-host-456",
		Hostname:   "test-hostname",
		EndpointID: "test-endpoint-789",
		Timestamp:  testTime,
		Metrics: []model.Metric{
			{
				Namespace:    "system",
				SubNamespace: "cpu",
				Name:         "usage_percent",
				Timestamp:    testTime,
				Value:        75.5,
				Unit:         "percent",
				Dimensions: map[string]string{
					"cpu":  "cpu0",
					"host": "test-host",
				},
			},
		},
		Meta: &model.Meta{
			AgentID:    "test-agent-123",
			HostID:     "test-host-456",
			Hostname:   "test-hostname",
			EndpointID: "test-endpoint-789",
			Kind:       "host",
			OS:         "linux",
		},
	}

	// Convert to OTLP
	otlpRequest := ConvertToOTLPMetrics(metricPayload)

	// Basic checks
	if otlpRequest == nil {
		t.Fatal("ConvertToOTLPMetrics returned nil")
	}

	if len(otlpRequest.ResourceMetrics) != 1 {
		t.Fatalf("Expected 1 ResourceMetrics, got %d", len(otlpRequest.ResourceMetrics))
	}

	resourceMetric := otlpRequest.ResourceMetrics[0]
	if resourceMetric.Resource == nil {
		t.Fatal("Resource is nil")
	}

	// Check metrics
	if len(resourceMetric.ScopeMetrics) != 1 {
		t.Fatalf("Expected 1 ScopeMetrics, got %d", len(resourceMetric.ScopeMetrics))
	}

	scopeMetric := resourceMetric.ScopeMetrics[0]
	if scopeMetric.Scope.Name != "system.cpu" {
		t.Errorf("Expected scope name 'system.cpu', got '%s'", scopeMetric.Scope.Name)
	}

	if len(scopeMetric.Metrics) != 1 {
		t.Fatalf("Expected 1 Metric, got %d", len(scopeMetric.Metrics))
	}

	metric := scopeMetric.Metrics[0]
	if metric.Name != "usage_percent" {
		t.Errorf("Expected metric name 'usage_percent', got '%s'", metric.Name)
	}

	if metric.Unit != "percent" {
		t.Errorf("Expected unit 'percent', got '%s'", metric.Unit)
	}
}

func TestConvertLogLevelToSeverity(t *testing.T) {
	tests := map[string]int32{
		"trace":   1,
		"TRACE":   1,
		"debug":   5,
		"DEBUG":   5,
		"info":    9,
		"INFO":    9,
		"warn":    13,
		"WARN":    13,
		"error":   17,
		"ERROR":   17,
		"fatal":   21,
		"FATAL":   21,
		"unknown": 0,
	}

	for level, expectedSeverity := range tests {
		severity := convertLogLevelToSeverity(level)
		if int32(severity) != expectedSeverity {
			t.Errorf("Level '%s': expected severity %d, got %d", level, expectedSeverity, int32(severity))
		}
	}
}
