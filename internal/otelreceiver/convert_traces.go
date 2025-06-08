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
	"time"

	"github.com/aaronlmathis/gosight-shared/model"

	otlpcoltrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// OTLPToTraceSpans converts an OTLP ExportTraceServiceRequest into GoSight’s []*model.TraceSpan.
func OTLPToTraceSpans(req *otlpcoltrace.ExportTraceServiceRequest) []*model.TraceSpan {
	var out []*model.TraceSpan

	for _, resourceSpans := range req.ResourceSpans {

		// Convert resource‐level attributes (e.g. service.name, host.name, etc.)
		resourceAttrs := convertKeyValueToStringMap(resourceSpans.Resource.Attributes)
		meta := buildMetaFromResourceAttrs(resourceAttrs)

		// Pull out ServiceName/HostID from resourceAttrs if present
		svcName := resourceAttrs["service.name"]
		hostID := resourceAttrs["host.name"] // or "host.id" depending on what you set

		for _, scopeSpans := range resourceSpans.ScopeSpans {
			// Optionally: instrumentation library name/version in scopeSpans.Scope.Name/Version

			for _, span := range scopeSpans.Spans {
				// 2) Build the basic TraceSpan
				ts := &model.TraceSpan{
					TraceID:      hex.EncodeToString(span.TraceId),
					SpanID:       hex.EncodeToString(span.SpanId),
					ParentSpanID: "", // fill below if present
					Name:         span.Name,

					ServiceName: svcName,
					HostID:      hostID,
					// If your agent sets EndpointID/AgentID at resource level:
					EndpointID: resourceAttrs["endpoint.id"],
					AgentID:    resourceAttrs["agent.id"],

					StartTime: time.Unix(0, int64(span.StartTimeUnixNano)),
					EndTime:   time.Unix(0, int64(span.EndTimeUnixNano)),
					// Duration in milliseconds:
					DurationMs: float64(span.EndTimeUnixNano-span.StartTimeUnixNano) / 1e6,

					StatusCode:    span.Status.GetCode().String(),
					StatusMessage: span.Status.GetMessage(),

					Attributes:    convertKeyValueToStringMap(span.Attributes),
					Events:        convertSpanEvents(span.Events),
					ResourceAttrs: resourceAttrs,
					Meta:          meta,
				}

				// ParentSpanID (if non‐zero length)
				if len(span.ParentSpanId) == 8 {
					ts.ParentSpanID = hex.EncodeToString(span.ParentSpanId)
				}

				out = append(out, ts)
			}
		}
	}

	return out
}

// convertSpanEvents converts OTLP SpanEvents into []model.SpanEvent.
func convertSpanEvents(ots []*tracepb.Span_Event) []model.SpanEvent {
	var out []model.SpanEvent
	for _, oe := range ots {
		ev := model.SpanEvent{
			Name:       oe.Name,
			Timestamp:  time.Unix(0, int64(oe.TimeUnixNano)),
			Attributes: convertKeyValueToStringMap(oe.Attributes),
		}
		out = append(out, ev)
	}
	return out
}

// ConvertSingleTraceSpanToOTLP converts a single *model.TraceSpan to an OTLP ExportTraceServiceRequest.
func ConvertSingleTraceSpanToOTLP(span *model.TraceSpan) *otlpcoltrace.ExportTraceServiceRequest {
	if span == nil {
		return nil
	}

	resource := convertMetaToResource(span.Meta)

	// Convert TraceID and SpanID from hex string to []byte
	decodeHex := func(hexStr string, wantLen int) []byte {
		if len(hexStr) != wantLen*2 {
			return nil
		}
		b, err := hex.DecodeString(hexStr)
		if err != nil || len(b) != wantLen {
			return nil
		}
		return b
	}

	otlpSpan := &tracepb.Span{
		TraceId:           decodeHex(span.TraceID, 16),
		SpanId:            decodeHex(span.SpanID, 8),
		Name:              span.Name,
		StartTimeUnixNano: uint64(span.StartTime.UnixNano()),
		EndTimeUnixNano:   uint64(span.EndTime.UnixNano()),
		Attributes:        convertStringMapToKeyValue(span.Attributes),
		Status: &tracepb.Status{
			Code:    parseStatusCode(span.StatusCode),
			Message: span.StatusMessage,
		},
	}
	if span.ParentSpanID != "" {
		otlpSpan.ParentSpanId = decodeHex(span.ParentSpanID, 8)
	}
	// Events
	for _, ev := range span.Events {
		otlpSpan.Events = append(otlpSpan.Events, &tracepb.Span_Event{
			Name:         ev.Name,
			TimeUnixNano: uint64(ev.Timestamp.UnixNano()),
			Attributes:   convertStringMapToKeyValue(ev.Attributes),
		})
	}

	// ScopeSpans: use ServiceName as scope name if present
	scopeName := span.ServiceName
	if scopeName == "" {
		scopeName = "gosight"
	}

	scopeSpans := []*tracepb.ScopeSpans{
		{
			Scope: &commonpb.InstrumentationScope{
				Name: scopeName,
			},
			Spans: []*tracepb.Span{otlpSpan},
		},
	}

	resourceSpans := []*tracepb.ResourceSpans{
		{
			Resource:   resource,
			ScopeSpans: scopeSpans,
		},
	}

	return &otlpcoltrace.ExportTraceServiceRequest{
		ResourceSpans: resourceSpans,
	}
}

// parseStatusCode maps string status code to OTLP Status_Code enum.
func parseStatusCode(code string) tracepb.Status_StatusCode {
	switch code {
	case "STATUS_CODE_OK", "OK":
		return tracepb.Status_STATUS_CODE_OK
	case "STATUS_CODE_ERROR", "ERROR":
		return tracepb.Status_STATUS_CODE_ERROR
	case "STATUS_CODE_UNSET", "UNSET":
		return tracepb.Status_STATUS_CODE_UNSET
	default:
		return tracepb.Status_STATUS_CODE_UNSET
	}
}
