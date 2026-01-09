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

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	opentelemetrylogger "github.com/casbin/casbin-opentelemetry-logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

func main() {
	// Create a Prometheus registry
	reg := prometheus.NewRegistry()

	// Create a Prometheus exporter for OpenTelemetry metrics
	exporter, err := promexporter.New(promexporter.WithRegisterer(reg))
	if err != nil {
		log.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	// Create a meter provider with the Prometheus exporter
	provider := metric.NewMeterProvider(metric.WithReader(exporter))

	// Create a new OpenTelemetryLogger with the meter provider
	logger := opentelemetrylogger.NewOpenTelemetryLoggerWithMeterProvider(provider)

	// Optional: Configure which event types to log
	// If not set, all event types will be logged
	err = logger.SetEventTypes([]opentelemetrylogger.EventType{
		opentelemetrylogger.EventEnforce,
		opentelemetrylogger.EventAddPolicy,
		opentelemetrylogger.EventRemovePolicy,
	})
	if err != nil {
		log.Fatalf("Failed to set event types: %v", err)
	}

	// Optional: Set a custom callback for additional processing
	err = logger.SetLogCallback(func(entry *opentelemetrylogger.LogEntry) error {
		fmt.Printf("Event: %s, Duration: %v\n", entry.EventType, entry.Duration)
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to set callback: %v", err)
	}

	// Simulate some enforce events
	simulateEnforceEvents(logger)

	// Simulate some policy operations
	simulatePolicyEvents(logger)

	// Start HTTP server to expose metrics
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	fmt.Println("Starting metrics server on :8080")
	fmt.Println("Visit http://localhost:8080/metrics to see the metrics")

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Failed to start metrics server: %v", err)
		}
	}()

	// Keep the example running for demonstration
	fmt.Println("\nPress Ctrl+C to stop...")
	select {}
}

func simulateEnforceEvents(logger *opentelemetrylogger.OpenTelemetryLogger) {
	fmt.Println("\n=== Simulating Enforce Events ===")

	scenarios := []struct {
		subject string
		object  string
		action  string
		domain  string
		allowed bool
	}{
		{"alice", "data1", "read", "domain1", true},
		{"alice", "data2", "write", "domain1", false},
		{"bob", "data1", "read", "domain2", true},
		{"bob", "data2", "write", "domain2", true},
		{"charlie", "data1", "delete", "", false},
	}

	for _, scenario := range scenarios {
		entry := &opentelemetrylogger.LogEntry{
			EventType: opentelemetrylogger.EventEnforce,
			Subject:   scenario.subject,
			Object:    scenario.object,
			Action:    scenario.action,
			Domain:    scenario.domain,
		}

		// Before event
		logger.OnBeforeEvent(entry)

		// Simulate processing time
		time.Sleep(10 * time.Millisecond)

		// After event
		entry.Allowed = scenario.allowed
		logger.OnAfterEvent(entry)

		fmt.Printf("Logged: %s %s %s (allowed: %v)\n",
			scenario.subject, scenario.action, scenario.object, scenario.allowed)
	}
}

func simulatePolicyEvents(logger *opentelemetrylogger.OpenTelemetryLogger) {
	fmt.Println("\n=== Simulating Policy Events ===")

	// Add policy event
	addEntry := &opentelemetrylogger.LogEntry{
		EventType: opentelemetrylogger.EventAddPolicy,
		RuleCount: 5,
	}
	logger.OnBeforeEvent(addEntry)
	time.Sleep(5 * time.Millisecond)
	logger.OnAfterEvent(addEntry)
	fmt.Println("Logged: AddPolicy (5 rules)")

	// Remove policy event
	removeEntry := &opentelemetrylogger.LogEntry{
		EventType: opentelemetrylogger.EventRemovePolicy,
		RuleCount: 2,
	}
	logger.OnBeforeEvent(removeEntry)
	time.Sleep(3 * time.Millisecond)
	logger.OnAfterEvent(removeEntry)
	fmt.Println("Logged: RemovePolicy (2 rules)")

	// Load policy event (not configured in event types, should be filtered)
	loadEntry := &opentelemetrylogger.LogEntry{
		EventType: opentelemetrylogger.EventLoadPolicy,
		RuleCount: 100,
	}
	logger.OnBeforeEvent(loadEntry)
	time.Sleep(20 * time.Millisecond)
	logger.OnAfterEvent(loadEntry)
	fmt.Println("Logged: LoadPolicy (100 rules) - filtered out")
}
