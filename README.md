# casbin-opentelemetry-logger

[![Go Report Card](https://goreportcard.com/badge/github.com/casbin/casbin-opentelemetry-logger)](https://goreportcard.com/report/github.com/casbin/casbin-opentelemetry-logger)
[![Go](https://github.com/casbin/casbin-opentelemetry-logger/actions/workflows/ci.yml/badge.svg)](https://github.com/casbin/casbin-opentelemetry-logger/actions/workflows/ci.yml)
[![Coverage Status](https://codecov.io/gh/casbin/casbin-opentelemetry-logger/branch/master/graph/badge.svg)](https://codecov.io/gh/casbin/casbin-opentelemetry-logger)
[![GoDoc](https://godoc.org/github.com/casbin/casbin-opentelemetry-logger?status.svg)](https://godoc.org/github.com/casbin/casbin-opentelemetry-logger)
[![Release](https://img.shields.io/github/release/casbin/casbin-opentelemetry-logger.svg)](https://github.com/casbin/casbin-opentelemetry-logger/releases/latest)
[![Discord](https://img.shields.io/discord/1022748306096537660?logo=discord&label=discord&color=5865F2)](https://discord.gg/S5UjpzGZjN)

An OpenTelemetry logger implementation for [Casbin](https://github.com/casbin/casbin), providing event-driven metrics collection for authorization events.

## Features

- **Event-Driven Logging**: Implements the Casbin Logger interface with support for event-driven logging
- **OpenTelemetry Metrics**: Exports comprehensive metrics using the OpenTelemetry standard
- **Customizable Event Types**: Filter which event types to log
- **Custom Callbacks**: Add custom processing for log entries
- **Context Support**: Support for custom contexts for propagation and cancellation

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

### Basic Usage

```go
package main

import (
    "context"
    
    opentelemetrylogger "github.com/casbin/casbin-opentelemetry-logger"
    "go.opentelemetry.io/otel"
)

func main() {
    // Get a meter from your OpenTelemetry provider
    meter := otel.Meter("casbin")
    
    // Create logger
    logger, err := opentelemetrylogger.NewOpenTelemetryLogger(meter)
    if err != nil {
        panic(err)
    }
    
    // Use with Casbin
    // enforcer.SetLogger(logger)
}
```

### With Custom Context

```go
ctx := context.Background()
logger, err := opentelemetrylogger.NewOpenTelemetryLoggerWithContext(ctx, meter)
if err != nil {
    panic(err)
}
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

## Complete Example with OTLP Exporter

```go
package main

import (
    "context"
    "log"
    "time"

    opentelemetrylogger "github.com/casbin/casbin-opentelemetry-logger"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/resource"
    semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func main() {
    ctx := context.Background()

    // Create OTLP exporter
    exporter, err := otlpmetricgrpc.New(ctx,
        otlpmetricgrpc.WithEndpoint("localhost:4317"),
        otlpmetricgrpc.WithInsecure(),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create resource
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName("casbin-app"),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create meter provider
    provider := metric.NewMeterProvider(
        metric.WithReader(metric.NewPeriodicReader(exporter)),
        metric.WithResource(res),
    )
    otel.SetMeterProvider(provider)

    // Create logger
    meter := otel.Meter("casbin")
    logger, err := opentelemetrylogger.NewOpenTelemetryLogger(meter)
    if err != nil {
        log.Fatal(err)
    }

    // Use with Casbin enforcer
    // enforcer.SetLogger(logger)

    // Shutdown
    defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := provider.Shutdown(ctx); err != nil {
            log.Printf("Error shutting down meter provider: %v", err)
        }
    }()
}
```

## OpenTelemetry Collector Configuration

To collect metrics from your application, configure the OpenTelemetry Collector:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"
  logging:
    loglevel: debug

service:
  pipelines:
    metrics:
      receivers: [otlp]
      exporters: [prometheus, logging]
```

## Visualization with Prometheus and Grafana

1. **Configure OpenTelemetry Collector** to export metrics to Prometheus (see above)
2. **Configure Prometheus** to scrape the OpenTelemetry Collector endpoint
3. **Import Grafana Dashboard** using similar panels as the Prometheus logger project

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Related Projects

- [Casbin](https://github.com/casbin/casbin) - An authorization library that supports access control models
- [OpenTelemetry](https://opentelemetry.io/) - Observability framework for cloud-native software
- [casbin-prometheus-logger](https://github.com/casbin/casbin-prometheus-logger) - Prometheus logger for Casbin