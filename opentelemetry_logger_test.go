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
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
)

func TestNewOpenTelemetryLogger(t *testing.T) {
	logger := NewOpenTelemetryLogger()
	if logger == nil {
		t.Fatal("NewOpenTelemetryLogger returned nil")
	}

	if logger.enabledEventTypes == nil {
		t.Error("enabledEventTypes map not initialized")
	}

	if logger.meter == nil {
		t.Error("meter not initialized")
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

func TestNewOpenTelemetryLoggerWithMeterProvider(t *testing.T) {
	provider := metric.NewMeterProvider()
	logger := NewOpenTelemetryLoggerWithMeterProvider(provider)

	if logger == nil {
		t.Fatal("NewOpenTelemetryLoggerWithMeterProvider returned nil")
	}

	if logger.enabledEventTypes == nil {
		t.Error("enabledEventTypes map not initialized")
	}

	if logger.meter == nil {
		t.Error("meter not initialized")
	}

	if logger.enforceDuration == nil {
		t.Error("enforceDuration not initialized")
	}
	if logger.enforceTotal == nil {
		t.Error("enforceTotal not initialized")
	}
	if logger.policyOpsTotal == nil {
		t.Error("policyOpsTotal not initialized")
	}
	if logger.policyOpsDuration == nil {
		t.Error("policyOpsDuration not initialized")
	}
	if logger.policyRulesCount == nil {
		t.Error("policyRulesCount not initialized")
	}
}

func TestSetEventTypes(t *testing.T) {
	logger := NewOpenTelemetryLogger()

	eventTypes := []EventType{EventEnforce, EventAddPolicy}
	err := logger.SetEventTypes(eventTypes)
	if err != nil {
		t.Fatalf("SetEventTypes failed: %v", err)
	}

	if !logger.enabledEventTypes[EventEnforce] {
		t.Error("EventEnforce not enabled")
	}
	if !logger.enabledEventTypes[EventAddPolicy] {
		t.Error("EventAddPolicy not enabled")
	}
	if logger.enabledEventTypes[EventRemovePolicy] {
		t.Error("EventRemovePolicy should not be enabled")
	}
}

func TestOnBeforeEvent_WithEnabledEvent(t *testing.T) {
	logger := NewOpenTelemetryLogger()
	logger.SetEventTypes([]EventType{EventEnforce})

	entry := &LogEntry{
		EventType: EventEnforce,
	}

	err := logger.OnBeforeEvent(entry)
	if err != nil {
		t.Fatalf("OnBeforeEvent failed: %v", err)
	}

	if !entry.IsActive {
		t.Error("Entry should be active for enabled event type")
	}

	if entry.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
}

func TestOnBeforeEvent_WithDisabledEvent(t *testing.T) {
	logger := NewOpenTelemetryLogger()
	logger.SetEventTypes([]EventType{EventEnforce})

	entry := &LogEntry{
		EventType: EventAddPolicy,
	}

	err := logger.OnBeforeEvent(entry)
	if err != nil {
		t.Fatalf("OnBeforeEvent failed: %v", err)
	}

	if entry.IsActive {
		t.Error("Entry should not be active for disabled event type")
	}
}

func TestOnBeforeEvent_WithNoFilter(t *testing.T) {
	logger := NewOpenTelemetryLogger()

	entry := &LogEntry{
		EventType: EventEnforce,
	}

	err := logger.OnBeforeEvent(entry)
	if err != nil {
		t.Fatalf("OnBeforeEvent failed: %v", err)
	}

	if !entry.IsActive {
		t.Error("Entry should be active when no event filter is set")
	}
}

func TestOnAfterEvent_InactiveEntry(t *testing.T) {
	logger := NewOpenTelemetryLogger()

	entry := &LogEntry{
		EventType: EventEnforce,
		IsActive:  false,
	}

	err := logger.OnAfterEvent(entry)
	if err != nil {
		t.Fatalf("OnAfterEvent failed: %v", err)
	}

	if entry.Duration != 0 {
		t.Error("Duration should not be set for inactive entry")
	}
}

func TestOnAfterEvent_EnforceEvent(t *testing.T) {
	provider := metric.NewMeterProvider()
	logger := NewOpenTelemetryLoggerWithMeterProvider(provider)

	entry := &LogEntry{
		EventType: EventEnforce,
		IsActive:  true,
		StartTime: time.Now().Add(-100 * time.Millisecond),
		Subject:   "alice",
		Object:    "data1",
		Action:    "read",
		Domain:    "domain1",
		Allowed:   true,
	}

	err := logger.OnAfterEvent(entry)
	if err != nil {
		t.Fatalf("OnAfterEvent failed: %v", err)
	}

	if entry.Duration == 0 {
		t.Error("Duration should be calculated")
	}

	if entry.EndTime.IsZero() {
		t.Error("EndTime should be set")
	}
}

func TestOnAfterEvent_PolicyEvent(t *testing.T) {
	provider := metric.NewMeterProvider()
	logger := NewOpenTelemetryLoggerWithMeterProvider(provider)

	entry := &LogEntry{
		EventType: EventAddPolicy,
		IsActive:  true,
		StartTime: time.Now().Add(-50 * time.Millisecond),
		RuleCount: 5,
	}

	err := logger.OnAfterEvent(entry)
	if err != nil {
		t.Fatalf("OnAfterEvent failed: %v", err)
	}

	if entry.Duration == 0 {
		t.Error("Duration should be calculated")
	}
}

func TestOnAfterEvent_WithError(t *testing.T) {
	provider := metric.NewMeterProvider()
	logger := NewOpenTelemetryLoggerWithMeterProvider(provider)

	entry := &LogEntry{
		EventType: EventAddPolicy,
		IsActive:  true,
		StartTime: time.Now().Add(-50 * time.Millisecond),
		RuleCount: 5,
		Error:     errors.New("test error"),
	}

	err := logger.OnAfterEvent(entry)
	if err != nil {
		t.Fatalf("OnAfterEvent failed: %v", err)
	}
}

func TestSetLogCallback(t *testing.T) {
	logger := NewOpenTelemetryLogger()

	callbackCalled := false
	callback := func(entry *LogEntry) error {
		callbackCalled = true
		return nil
	}

	err := logger.SetLogCallback(callback)
	if err != nil {
		t.Fatalf("SetLogCallback failed: %v", err)
	}

	entry := &LogEntry{
		EventType: EventEnforce,
		IsActive:  true,
		StartTime: time.Now(),
	}

	logger.OnAfterEvent(entry)

	if !callbackCalled {
		t.Error("Callback should have been called")
	}
}

func TestSetLogCallback_WithError(t *testing.T) {
	logger := NewOpenTelemetryLogger()

	expectedError := errors.New("callback error")
	callback := func(entry *LogEntry) error {
		return expectedError
	}

	err := logger.SetLogCallback(callback)
	if err != nil {
		t.Fatalf("SetLogCallback failed: %v", err)
	}

	entry := &LogEntry{
		EventType: EventEnforce,
		IsActive:  true,
		StartTime: time.Now(),
	}

	err = logger.OnAfterEvent(entry)
	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}
}

func TestGetMeter(t *testing.T) {
	logger := NewOpenTelemetryLogger()

	meter := logger.GetMeter()
	if meter == nil {
		t.Error("GetMeter should return non-nil meter")
	}
}

func TestEnforceMetrics_DefaultDomain(t *testing.T) {
	provider := metric.NewMeterProvider()
	logger := NewOpenTelemetryLoggerWithMeterProvider(provider)

	entry := &LogEntry{
		EventType: EventEnforce,
		IsActive:  true,
		StartTime: time.Now(),
		Subject:   "alice",
		Object:    "data1",
		Action:    "read",
		Domain:    "", // Empty domain should default to "default"
		Allowed:   true,
	}

	err := logger.OnAfterEvent(entry)
	if err != nil {
		t.Fatalf("OnAfterEvent failed: %v", err)
	}
}

func TestAllEventTypes(t *testing.T) {
	provider := metric.NewMeterProvider()
	logger := NewOpenTelemetryLoggerWithMeterProvider(provider)

	eventTypes := []EventType{
		EventEnforce,
		EventAddPolicy,
		EventRemovePolicy,
		EventLoadPolicy,
		EventSavePolicy,
	}

	for _, eventType := range eventTypes {
		entry := &LogEntry{
			EventType: eventType,
			IsActive:  true,
			StartTime: time.Now().Add(-10 * time.Millisecond),
			RuleCount: 3,
			Allowed:   true,
		}

		logger.OnBeforeEvent(entry)
		err := logger.OnAfterEvent(entry)
		if err != nil {
			t.Fatalf("OnAfterEvent failed for %s: %v", eventType, err)
		}

		if entry.Duration == 0 {
			t.Errorf("Duration should be calculated for %s", eventType)
		}
	}
}
