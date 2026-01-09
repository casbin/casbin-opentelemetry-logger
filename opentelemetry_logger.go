// Copyright 2026 The casbin Authors. All Rights Reserved.
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

package opentelemetrylogger

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// OpenTelemetryLogger is a logger that exports metrics to OpenTelemetry.
type OpenTelemetryLogger struct {
	enabledEventTypes map[EventType]bool
	callback          func(entry *LogEntry) error

	// OpenTelemetry metrics
	enforceDuration   metric.Float64Histogram
	enforceTotal      metric.Int64Counter
	policyOpsTotal    metric.Int64Counter
	policyOpsDuration metric.Float64Histogram
	policyRulesCount  metric.Int64Gauge

	ctx context.Context
}

// NewOpenTelemetryLogger creates a new OpenTelemetryLogger with the provided meter.
func NewOpenTelemetryLogger(meter metric.Meter) (*OpenTelemetryLogger, error) {
	return NewOpenTelemetryLoggerWithContext(context.Background(), meter)
}

// NewOpenTelemetryLoggerWithContext creates a new OpenTelemetryLogger with a custom context and meter.
func NewOpenTelemetryLoggerWithContext(ctx context.Context, meter metric.Meter) (*OpenTelemetryLogger, error) {
	logger := &OpenTelemetryLogger{
		enabledEventTypes: make(map[EventType]bool),
		ctx:               ctx,
	}

	var err error

	// Create enforce duration histogram
	logger.enforceDuration, err = meter.Float64Histogram(
		"casbin.enforce.duration",
		metric.WithDescription("Duration of enforce requests in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// Create enforce total counter
	logger.enforceTotal, err = meter.Int64Counter(
		"casbin.enforce.total",
		metric.WithDescription("Total number of enforce requests"),
	)
	if err != nil {
		return nil, err
	}

	// Create policy operations total counter
	logger.policyOpsTotal, err = meter.Int64Counter(
		"casbin.policy.operations.total",
		metric.WithDescription("Total number of policy operations"),
	)
	if err != nil {
		return nil, err
	}

	// Create policy operations duration histogram
	logger.policyOpsDuration, err = meter.Float64Histogram(
		"casbin.policy.operations.duration",
		metric.WithDescription("Duration of policy operations in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// Create policy rules count gauge
	logger.policyRulesCount, err = meter.Int64Gauge(
		"casbin.policy.rules.count",
		metric.WithDescription("Number of policy rules affected by operations"),
	)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// SetEventTypes configures which event types should be logged.
func (l *OpenTelemetryLogger) SetEventTypes(eventTypes []EventType) error {
	l.enabledEventTypes = make(map[EventType]bool)
	for _, eventType := range eventTypes {
		l.enabledEventTypes[eventType] = true
	}
	return nil
}

// OnBeforeEvent is called before an event occurs.
func (l *OpenTelemetryLogger) OnBeforeEvent(entry *LogEntry) error {
	if len(l.enabledEventTypes) > 0 && !l.enabledEventTypes[entry.EventType] {
		entry.IsActive = false
		return nil
	}

	entry.IsActive = true
	entry.StartTime = time.Now()
	return nil
}

// OnAfterEvent is called after an event completes and records metrics.
func (l *OpenTelemetryLogger) OnAfterEvent(entry *LogEntry) error {
	if !entry.IsActive {
		return nil
	}

	entry.EndTime = time.Now()
	entry.Duration = entry.EndTime.Sub(entry.StartTime)

	// Record metrics based on event type
	switch entry.EventType {
	case EventEnforce:
		l.recordEnforceMetrics(entry)
	case EventAddPolicy, EventRemovePolicy, EventLoadPolicy, EventSavePolicy:
		l.recordPolicyMetrics(entry)
	}

	// Call custom callback if set
	if l.callback != nil {
		return l.callback(entry)
	}

	return nil
}

// SetLogCallback sets a custom callback function for log entries.
func (l *OpenTelemetryLogger) SetLogCallback(callback func(entry *LogEntry) error) error {
	l.callback = callback
	return nil
}

// recordEnforceMetrics records metrics for enforce events.
func (l *OpenTelemetryLogger) recordEnforceMetrics(entry *LogEntry) {
	domain := entry.Domain
	if domain == "" {
		domain = "default"
	}

	allowed := "false"
	if entry.Allowed {
		allowed = "true"
	}

	attrs := []attribute.KeyValue{
		attribute.String("allowed", allowed),
		attribute.String("domain", domain),
	}

	l.enforceDuration.Record(l.ctx, entry.Duration.Seconds(), metric.WithAttributes(attrs...))
	l.enforceTotal.Add(l.ctx, 1, metric.WithAttributes(attrs...))
}

// recordPolicyMetrics records metrics for policy operation events.
func (l *OpenTelemetryLogger) recordPolicyMetrics(entry *LogEntry) {
	operation := string(entry.EventType)
	success := "true"
	if entry.Error != nil {
		success = "false"
	}

	opsAttrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.String("success", success),
	}

	durationAttrs := []attribute.KeyValue{
		attribute.String("operation", operation),
	}

	l.policyOpsTotal.Add(l.ctx, 1, metric.WithAttributes(opsAttrs...))
	l.policyOpsDuration.Record(l.ctx, entry.Duration.Seconds(), metric.WithAttributes(durationAttrs...))

	if entry.RuleCount > 0 {
		countAttrs := []attribute.KeyValue{
			attribute.String("operation", operation),
		}
		l.policyRulesCount.Record(l.ctx, int64(entry.RuleCount), metric.WithAttributes(countAttrs...))
	}
}

// GetEnforceDuration returns the enforce duration histogram metric.
func (l *OpenTelemetryLogger) GetEnforceDuration() metric.Float64Histogram {
	return l.enforceDuration
}

// GetEnforceTotal returns the enforce total counter metric.
func (l *OpenTelemetryLogger) GetEnforceTotal() metric.Int64Counter {
	return l.enforceTotal
}

// GetPolicyOpsTotal returns the policy operations total counter metric.
func (l *OpenTelemetryLogger) GetPolicyOpsTotal() metric.Int64Counter {
	return l.policyOpsTotal
}

// GetPolicyOpsDuration returns the policy operations duration histogram metric.
func (l *OpenTelemetryLogger) GetPolicyOpsDuration() metric.Float64Histogram {
	return l.policyOpsDuration
}

// GetPolicyRulesCount returns the policy rules count gauge metric.
func (l *OpenTelemetryLogger) GetPolicyRulesCount() metric.Int64Gauge {
	return l.policyRulesCount
}
