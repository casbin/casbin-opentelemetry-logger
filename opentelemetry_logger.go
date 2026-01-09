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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	meterName = "github.com/casbin/casbin-opentelemetry-logger"
)

// OpenTelemetryLogger is a logger that exports metrics to OpenTelemetry.
type OpenTelemetryLogger struct {
	enabledEventTypes map[EventType]bool
	callback          func(entry *LogEntry) error

	meter metric.Meter

	// OpenTelemetry metrics
	enforceDuration    metric.Float64Histogram
	enforceTotal       metric.Int64Counter
	policyOpsTotal     metric.Int64Counter
	policyOpsDuration  metric.Float64Histogram
	policyRulesCount   metric.Int64Gauge
}

// NewOpenTelemetryLogger creates a new OpenTelemetryLogger with default meter provider.
func NewOpenTelemetryLogger() *OpenTelemetryLogger {
	return NewOpenTelemetryLoggerWithMeterProvider(otel.GetMeterProvider())
}

// NewOpenTelemetryLoggerWithMeterProvider creates a new OpenTelemetryLogger with a custom meter provider.
func NewOpenTelemetryLoggerWithMeterProvider(provider metric.MeterProvider) *OpenTelemetryLogger {
	meter := provider.Meter(meterName)

	logger := &OpenTelemetryLogger{
		enabledEventTypes: make(map[EventType]bool),
		meter:             meter,
	}

	// Initialize metrics
	var err error

	logger.enforceDuration, err = meter.Float64Histogram(
		"casbin.enforce.duration",
		metric.WithDescription("Duration of enforce requests in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	logger.enforceTotal, err = meter.Int64Counter(
		"casbin.enforce.total",
		metric.WithDescription("Total number of enforce requests"),
	)
	if err != nil {
		panic(err)
	}

	logger.policyOpsTotal, err = meter.Int64Counter(
		"casbin.policy.operations.total",
		metric.WithDescription("Total number of policy operations"),
	)
	if err != nil {
		panic(err)
	}

	logger.policyOpsDuration, err = meter.Float64Histogram(
		"casbin.policy.operations.duration",
		metric.WithDescription("Duration of policy operations in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	logger.policyRulesCount, err = meter.Int64Gauge(
		"casbin.policy.rules.count",
		metric.WithDescription("Number of policy rules affected by operations"),
	)
	if err != nil {
		panic(err)
	}

	return logger
}

// SetEventTypes configures which event types should be logged.
func (o *OpenTelemetryLogger) SetEventTypes(eventTypes []EventType) error {
	o.enabledEventTypes = make(map[EventType]bool)
	for _, eventType := range eventTypes {
		o.enabledEventTypes[eventType] = true
	}
	return nil
}

// OnBeforeEvent is called before an event occurs.
func (o *OpenTelemetryLogger) OnBeforeEvent(entry *LogEntry) error {
	if len(o.enabledEventTypes) > 0 && !o.enabledEventTypes[entry.EventType] {
		entry.IsActive = false
		return nil
	}

	entry.IsActive = true
	entry.StartTime = time.Now()
	return nil
}

// OnAfterEvent is called after an event completes and records metrics.
func (o *OpenTelemetryLogger) OnAfterEvent(entry *LogEntry) error {
	if !entry.IsActive {
		return nil
	}

	entry.EndTime = time.Now()
	entry.Duration = entry.EndTime.Sub(entry.StartTime)

	// Record metrics based on event type
	switch entry.EventType {
	case EventEnforce:
		o.recordEnforceMetrics(entry)
	case EventAddPolicy, EventRemovePolicy, EventLoadPolicy, EventSavePolicy:
		o.recordPolicyMetrics(entry)
	}

	// Call custom callback if set
	if o.callback != nil {
		return o.callback(entry)
	}

	return nil
}

// SetLogCallback sets a custom callback function for log entries.
func (o *OpenTelemetryLogger) SetLogCallback(callback func(entry *LogEntry) error) error {
	o.callback = callback
	return nil
}

// recordEnforceMetrics records metrics for enforce events.
func (o *OpenTelemetryLogger) recordEnforceMetrics(entry *LogEntry) {
	domain := entry.Domain
	if domain == "" {
		domain = "default"
	}

	allowed := "false"
	if entry.Allowed {
		allowed = "true"
	}

	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("allowed", allowed),
		attribute.String("domain", domain),
	}

	o.enforceDuration.Record(ctx, entry.Duration.Seconds(), metric.WithAttributes(attrs...))
	o.enforceTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// recordPolicyMetrics records metrics for policy operation events.
func (o *OpenTelemetryLogger) recordPolicyMetrics(entry *LogEntry) {
	operation := string(entry.EventType)
	success := "true"
	if entry.Error != nil {
		success = "false"
	}

	ctx := context.Background()

	opAttrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.String("success", success),
	}

	o.policyOpsTotal.Add(ctx, 1, metric.WithAttributes(opAttrs...))

	durationAttrs := []attribute.KeyValue{
		attribute.String("operation", operation),
	}
	o.policyOpsDuration.Record(ctx, entry.Duration.Seconds(), metric.WithAttributes(durationAttrs...))

	if entry.RuleCount > 0 {
		o.policyRulesCount.Record(ctx, int64(entry.RuleCount), metric.WithAttributes(durationAttrs...))
	}
}

// GetMeter returns the OpenTelemetry meter used by this logger.
func (o *OpenTelemetryLogger) GetMeter() metric.Meter {
	return o.meter
}
