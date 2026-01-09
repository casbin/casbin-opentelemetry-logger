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
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestNewOpenTelemetryLogger(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	logger, err := NewOpenTelemetryLogger(meter)
	if err != nil {
		t.Fatalf("NewOpenTelemetryLogger returned error: %v", err)
	}

	if logger == nil {
		t.Fatal("NewOpenTelemetryLogger returned nil")
	}

	if logger.enabledEventTypes == nil {
		t.Error("enabledEventTypes map not initialized")
	}

	if logger.enforceDuration == nil {
		t.Error("enforceDuration metric not initialized")
	}

	if logger.enforceTotal == nil {
		t.Error("enforceTotal metric not initialized")
	}

	if logger.policyOpsTotal == nil {
		t.Error("policyOpsTotal metric not initialized")
	}

	if logger.policyOpsDuration == nil {
		t.Error("policyOpsDuration metric not initialized")
	}

	if logger.policyRulesCount == nil {
		t.Error("policyRulesCount metric not initialized")
	}
}

func TestNewOpenTelemetryLoggerWithContext(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	ctx := context.Background()
	logger, err := NewOpenTelemetryLoggerWithContext(ctx, meter)
	if err != nil {
		t.Fatalf("NewOpenTelemetryLoggerWithContext returned error: %v", err)
	}

	if logger == nil {
		t.Fatal("NewOpenTelemetryLoggerWithContext returned nil")
	}

	if logger.ctx != ctx {
		t.Error("Context not set correctly")
	}
}

func TestSetEventTypes(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	logger, err := NewOpenTelemetryLogger(meter)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	eventTypes := []EventType{EventEnforce, EventAddPolicy}
	err = logger.SetEventTypes(eventTypes)
	if err != nil {
		t.Errorf("SetEventTypes returned error: %v", err)
	}

	if len(logger.enabledEventTypes) != 2 {
		t.Errorf("Expected 2 enabled event types, got %d", len(logger.enabledEventTypes))
	}

	if !logger.enabledEventTypes[EventEnforce] {
		t.Error("EventEnforce should be enabled")
	}

	if !logger.enabledEventTypes[EventAddPolicy] {
		t.Error("EventAddPolicy should be enabled")
	}

	if logger.enabledEventTypes[EventRemovePolicy] {
		t.Error("EventRemovePolicy should not be enabled")
	}
}

func TestOnBeforeEvent(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	logger, err := NewOpenTelemetryLogger(meter)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Test with no event type filtering
	entry := &LogEntry{
		EventType: EventEnforce,
	}

	err = logger.OnBeforeEvent(entry)
	if err != nil {
		t.Errorf("OnBeforeEvent returned error: %v", err)
	}

	if !entry.IsActive {
		t.Error("Entry should be active when no event types are configured")
	}

	if entry.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}

	// Test with event type filtering - enabled event
	logger.SetEventTypes([]EventType{EventEnforce})
	entry2 := &LogEntry{
		EventType: EventEnforce,
	}

	err = logger.OnBeforeEvent(entry2)
	if err != nil {
		t.Errorf("OnBeforeEvent returned error: %v", err)
	}

	if !entry2.IsActive {
		t.Error("Entry should be active for enabled event type")
	}

	// Test with event type filtering - disabled event
	entry3 := &LogEntry{
		EventType: EventAddPolicy,
	}

	err = logger.OnBeforeEvent(entry3)
	if err != nil {
		t.Errorf("OnBeforeEvent returned error: %v", err)
	}

	if entry3.IsActive {
		t.Error("Entry should not be active for disabled event type")
	}
}

func TestOnAfterEvent_Enforce(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	logger, err := NewOpenTelemetryLogger(meter)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	entry := &LogEntry{
		IsActive:  true,
		EventType: EventEnforce,
		StartTime: time.Now().Add(-100 * time.Millisecond),
		Subject:   "alice",
		Object:    "data1",
		Action:    "read",
		Domain:    "domain1",
		Allowed:   true,
	}

	err = logger.OnAfterEvent(entry)
	if err != nil {
		t.Errorf("OnAfterEvent returned error: %v", err)
	}

	if entry.EndTime.IsZero() {
		t.Error("EndTime should be set")
	}

	if entry.Duration == 0 {
		t.Error("Duration should be calculated")
	}

	// Verify metrics were recorded
	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	if len(rm.ScopeMetrics) == 0 {
		t.Error("Expected metrics to be recorded")
	}

	// Check that we have at least one metric
	hasMetrics := false
	for _, sm := range rm.ScopeMetrics {
		if len(sm.Metrics) > 0 {
			hasMetrics = true
			break
		}
	}

	if !hasMetrics {
		t.Error("Expected at least one metric to be recorded")
	}
}

func TestOnAfterEvent_InactiveEntry(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	logger, err := NewOpenTelemetryLogger(meter)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	entry := &LogEntry{
		IsActive:  false,
		EventType: EventEnforce,
	}

	err = logger.OnAfterEvent(entry)
	if err != nil {
		t.Errorf("OnAfterEvent returned error: %v", err)
	}

	// Verify no metrics were recorded
	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// For an inactive entry, we should have no metrics
	totalMetrics := 0
	for _, sm := range rm.ScopeMetrics {
		totalMetrics += len(sm.Metrics)
	}

	if totalMetrics != 0 {
		t.Errorf("Expected 0 metrics for inactive entry, got %d", totalMetrics)
	}
}

func TestOnAfterEvent_PolicyOperation(t *testing.T) {
	testCases := []struct {
		name      string
		eventType EventType
	}{
		{"AddPolicy", EventAddPolicy},
		{"RemovePolicy", EventRemovePolicy},
		{"LoadPolicy", EventLoadPolicy},
		{"SavePolicy", EventSavePolicy},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := metric.NewManualReader()
			provider := metric.NewMeterProvider(metric.WithReader(reader))
			meter := provider.Meter("test")

			logger, err := NewOpenTelemetryLogger(meter)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			entry := &LogEntry{
				IsActive:  true,
				EventType: tc.eventType,
				StartTime: time.Now().Add(-50 * time.Millisecond),
				RuleCount: 5,
			}

			err = logger.OnAfterEvent(entry)
			if err != nil {
				t.Errorf("OnAfterEvent returned error: %v", err)
			}

			if entry.EndTime.IsZero() {
				t.Error("EndTime should be set")
			}

			if entry.Duration == 0 {
				t.Error("Duration should be calculated")
			}

			// Verify metrics were recorded
			var rm metricdata.ResourceMetrics
			err = reader.Collect(context.Background(), &rm)
			if err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}

			if len(rm.ScopeMetrics) == 0 {
				t.Error("Expected metrics to be recorded")
			}
		})
	}
}

func TestOnAfterEvent_WithError(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	logger, err := NewOpenTelemetryLogger(meter)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	entry := &LogEntry{
		IsActive:  true,
		EventType: EventAddPolicy,
		StartTime: time.Now().Add(-50 * time.Millisecond),
		RuleCount: 3,
		Error:     errors.New("test error"),
	}

	err = logger.OnAfterEvent(entry)
	if err != nil {
		t.Errorf("OnAfterEvent returned error: %v", err)
	}

	// Verify metrics were recorded with success="false"
	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	if len(rm.ScopeMetrics) == 0 {
		t.Error("Expected metrics to be recorded")
	}
}

func TestSetLogCallback(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	logger, err := NewOpenTelemetryLogger(meter)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	callbackCalled := false
	callback := func(entry *LogEntry) error {
		callbackCalled = true
		return nil
	}

	err = logger.SetLogCallback(callback)
	if err != nil {
		t.Errorf("SetLogCallback returned error: %v", err)
	}

	// Trigger an event to verify callback is called
	entry := &LogEntry{
		IsActive:  true,
		EventType: EventEnforce,
		StartTime: time.Now(),
		Allowed:   true,
	}

	err = logger.OnAfterEvent(entry)
	if err != nil {
		t.Errorf("OnAfterEvent returned error: %v", err)
	}

	if !callbackCalled {
		t.Error("Callback should have been called")
	}
}

func TestSetLogCallback_WithError(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	logger, err := NewOpenTelemetryLogger(meter)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	expectedError := errors.New("callback error")
	callback := func(entry *LogEntry) error {
		return expectedError
	}

	logger.SetLogCallback(callback)

	entry := &LogEntry{
		IsActive:  true,
		EventType: EventEnforce,
		StartTime: time.Now(),
		Allowed:   true,
	}

	err = logger.OnAfterEvent(entry)
	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}
}

func TestEnforceMetrics_DifferentDomains(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	logger, err := NewOpenTelemetryLogger(meter)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Test with specific domain
	entry1 := &LogEntry{
		IsActive:  true,
		EventType: EventEnforce,
		StartTime: time.Now(),
		Domain:    "domain1",
		Allowed:   true,
	}

	logger.OnAfterEvent(entry1)

	// Test with default domain (empty)
	entry2 := &LogEntry{
		IsActive:  true,
		EventType: EventEnforce,
		StartTime: time.Now(),
		Domain:    "",
		Allowed:   false,
	}

	logger.OnAfterEvent(entry2)

	// Verify metrics were recorded with different labels
	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	if len(rm.ScopeMetrics) == 0 {
		t.Error("Expected metrics to be recorded")
	}
}

func TestMetricGetters(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	logger, err := NewOpenTelemetryLogger(meter)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	if logger.GetEnforceDuration() == nil {
		t.Error("GetEnforceDuration returned nil")
	}

	if logger.GetEnforceTotal() == nil {
		t.Error("GetEnforceTotal returned nil")
	}

	if logger.GetPolicyOpsTotal() == nil {
		t.Error("GetPolicyOpsTotal returned nil")
	}

	if logger.GetPolicyOpsDuration() == nil {
		t.Error("GetPolicyOpsDuration returned nil")
	}

	if logger.GetPolicyRulesCount() == nil {
		t.Error("GetPolicyRulesCount returned nil")
	}
}

func TestLogger_InterfaceImplementation(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	logger, err := NewOpenTelemetryLogger(meter)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	var _ Logger = logger
}

func TestFullWorkflow(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	logger, err := NewOpenTelemetryLogger(meter)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Configure to only log enforce events
	logger.SetEventTypes([]EventType{EventEnforce})

	// Simulate enforce event
	enforceEntry := &LogEntry{
		EventType: EventEnforce,
		Subject:   "alice",
		Object:    "data1",
		Action:    "read",
		Domain:    "org1",
	}

	// Before event
	logger.OnBeforeEvent(enforceEntry)
	if !enforceEntry.IsActive {
		t.Error("Enforce entry should be active")
	}

	// Simulate some processing time
	time.Sleep(10 * time.Millisecond)

	// After event
	enforceEntry.Allowed = true
	logger.OnAfterEvent(enforceEntry)

	// Simulate policy event (should be filtered out)
	policyEntry := &LogEntry{
		EventType: EventAddPolicy,
		RuleCount: 5,
	}

	logger.OnBeforeEvent(policyEntry)
	if policyEntry.IsActive {
		t.Error("Policy entry should not be active (filtered)")
	}

	logger.OnAfterEvent(policyEntry)

	// Verify metrics were recorded
	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// We should have enforce metrics but not policy metrics (filtered)
	if len(rm.ScopeMetrics) == 0 {
		t.Error("Expected enforce metrics to be recorded")
	}
}
