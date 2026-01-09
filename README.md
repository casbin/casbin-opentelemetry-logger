# casbin-opentelemetry-logger

[![Go Report Card](https://goreportcard.com/badge/github.com/casbin/casbin-opentelemetry-logger)](https://goreportcard.com/report/github.com/casbin/casbin-opentelemetry-logger)
[![GoDoc](https://godoc.org/github.com/casbin/casbin-opentelemetry-logger?status.svg)](https://godoc.org/github.com/casbin/casbin-opentelemetry-logger)
[![Release](https://img.shields.io/github/release/casbin/casbin-opentelemetry-logger.svg)](https://github.com/casbin/casbin-opentelemetry-logger/releases/latest)
[![Discord](https://img.shields.io/discord/1022748306096537660?logo=discord&label=discord&color=5865F2)](https://discord.gg/S5UjpzGZjN)

An OpenTelemetry logger implementation for [Casbin](https://github.com/casbin/casbin), providing event-driven metrics collection for authorization events.

## Features

- **Event-Driven Logging**: Implements the Casbin Logger interface with support for event-driven logging
- **OpenTelemetry Metrics**: Exports comprehensive metrics for Casbin operations using OpenTelemetry
- **Customizable Event Types**: Filter which event types to log
- **Custom Callbacks**: Add custom processing for log entries
- **Multiple Exporters**: Support for any OpenTelemetry-compatible exporter (Prometheus, OTLP, etc.)

## Metrics Exported

### Enforce Metrics

- `casbin.enforce.total` - Total number of enforce requests (labeled by `allowed`, `domain`)
- `casbin.enforce.duration` - Duration of enforce requests in seconds (labeled by `allowed`, `domain`)

### Policy Operation Metrics

- `casbin.policy.operations.total` - Total number of policy operations (labeled by `operation`, `success`)
- `casbin.policy.operations.duration` - Duration of policy operations in seconds (labeled by `operation`)
- `casbin.policy.rules.count` - Number of policy rules affected by operations (labeled by `operation`)

## Installation

```bash
go get github.com/casbin/casbin-opentelemetry-logger
```

## Usage

### Basic Usage with Prometheus Exporter

```go
package main

import (
    "log"
    "net/http"
    
    opentelemetrylogger "github.com/casbin/casbin-opentelemetry-logger"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/sdk/metric"
)

func main() {
    // Create a Prometheus registry
    reg := prometheus.NewRegistry()
    
    // Create a Prometheus exporter for OpenTelemetry metrics
    exporter, err := prometheus.New(prometheus.WithRegisterer(reg))
    if err != nil {
        log.Fatalf("Failed to create Prometheus exporter: %v", err)
    }
    
    // Create a meter provider with the Prometheus exporter
    provider := metric.NewMeterProvider(metric.WithReader(exporter))
    
    // Create a new OpenTelemetryLogger with the meter provider
    logger := opentelemetrylogger.NewOpenTelemetryLoggerWithMeterProvider(provider)
    
    // Use with Casbin
    // enforcer.SetLogger(logger)
    
    // Expose metrics endpoint
    http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
    http.ListenAndServe(":8080", nil)
}
```

### Using Default Meter Provider

```go
// Create logger with default global meter provider
logger := opentelemetrylogger.NewOpenTelemetryLogger()
```

### Configure Event Types

```go
// Only log specific event types
logger.SetEventTypes([]opentelemetrylogger.EventType{
    opentelemetrylogger.EventEnforce,
    opentelemetrylogger.EventAddPolicy,
})
```

### Add Custom Callback

```go
// Add custom processing for log entries
logger.SetLogCallback(func(entry *opentelemetrylogger.LogEntry) error {
    fmt.Printf("Event: %s, Duration: %v\n", entry.EventType, entry.Duration)
    return nil
})
```

## Event Types

The logger supports the following event types:

- `EventEnforce` - Authorization enforcement requests
- `EventAddPolicy` - Policy addition operations
- `EventRemovePolicy` - Policy removal operations
- `EventLoadPolicy` - Policy loading operations
- `EventSavePolicy` - Policy saving operations

## Using with Different Exporters

### OTLP Exporter

```go
import (
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
    "go.opentelemetry.io/otel/sdk/metric"
)

exporter, err := otlpmetricgrpc.New(context.Background())
if err != nil {
    log.Fatal(err)
}

provider := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(exporter)))
logger := opentelemetrylogger.NewOpenTelemetryLoggerWithMeterProvider(provider)
```

### stdout Exporter (for testing)

```go
import (
    "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
    "go.opentelemetry.io/otel/sdk/metric"
)

exporter, err := stdoutmetric.New()
if err != nil {
    log.Fatal(err)
}

provider := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(exporter)))
logger := opentelemetrylogger.NewOpenTelemetryLoggerWithMeterProvider(provider)
```

## Example

See the [examples/basic](examples/basic/main.go) directory for a complete working example.

To run the example:

```bash
cd examples/basic
go run main.go
```

Then visit [http://localhost:8080/metrics](http://localhost:8080/metrics) to see the exported metrics.

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Related Projects

- [Casbin](https://github.com/casbin/casbin) - An authorization library that supports access control models
- [OpenTelemetry](https://opentelemetry.io/) - Observability framework for cloud-native software
- [casbin-prometheus-logger](https://github.com/casbin/casbin-prometheus-logger) - Prometheus logger for Casbin