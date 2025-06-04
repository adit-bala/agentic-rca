# ServiceGraph Helm Chart

A Helm chart for distributed tracing with Grafana Beyla and OpenTelemetry Collector.

## Overview

This chart deploys:
- **Grafana Beyla** as a DaemonSet configured for distributed tracing only
- **OpenTelemetry Collector** as a Deployment to process and export traces from Beyla
- **ServiceGraph Builder** as a Deployment to receive and process traces via OTLP gRPC

## Features

- ✅ Beyla deployed as DaemonSet for node-level eBPF instrumentation
- ✅ Beyla configured for distributed tracing only (no metrics)
- ✅ OpenTelemetry Collector with OTLP gRPC export
- ✅ ServiceGraph Builder for trace processing and analysis
- ✅ Automatic integration between OTel Collector and ServiceGraph Builder
- ✅ Minimal configuration surface in values.yaml
- ✅ Support for authentication via secrets or headers
- ✅ Kubernetes service discovery and filtering
- ✅ RBAC configuration for Beyla

## Installation

```bash
# Install with default values (includes ServiceGraph Builder)
helm install my-servicegraph ./servicegraph-helm

# Install with custom namespace
helm install my-servicegraph ./servicegraph-helm \
  --set global.namespace=observability

# Install with OTLP gRPC export to external endpoint (bypasses ServiceGraph Builder)
helm install my-servicegraph ./servicegraph-helm \
  --set otelCollector.export.endpoint=http://tempo:4317

# Install with ServiceGraph Builder disabled
helm install my-servicegraph ./servicegraph-helm \
  --set servicegraphBuilder.enabled=false
```

## Configuration

### Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.namespace` | Namespace to deploy components | `default` |
| `beyla.enabled` | Enable Beyla DaemonSet | `true` |
| `beyla.discovery.namespace` | K8s namespace to monitor | `"."` |
| `otelCollector.enabled` | Enable OTel Collector | `true` |
| `otelCollector.export.endpoint` | OTLP gRPC endpoint URL (overrides ServiceGraph Builder) | `""` |
| `otelCollector.export.secretName` | Secret containing auth token | `""` |
| `otelCollector.export.headers` | Authentication headers | `{}` |
| `servicegraphBuilder.enabled` | Enable ServiceGraph Builder | `true` |
| `servicegraphBuilder.service.port` | ServiceGraph Builder service port | `8083` |

### Example: Export to Tempo with authentication

```yaml
# values.yaml
global:
  namespace: observability

otelCollector:
  export:
    endpoint: http://tempo:4317
    headers:
      authorization: "Bearer your-token"
```

### Example: Export with secret-based authentication

```yaml
# values.yaml
otelCollector:
  export:
    endpoint: http://tempo:4317
    secretName: tempo-auth-secret
```

Create the secret:
```bash
kubectl create secret generic tempo-auth-secret \
  --from-literal=token=your-bearer-token
```

## Architecture

```
┌─────────────────┐    OTLP/HTTP    ┌─────────────────┐    OTLP/gRPC     ┌─────────────────┐
│                 │    Port 4318    │                 │                  │                 │
│  Grafana Beyla  ├────────────────►│ OTel Collector  ├─────────────────►│ Trace Backends  │
│   (DaemonSet)   │                 │   (Deployment)  │                  │   (Tempo/etc)   │
└─────────────────┘                 └─────────────────┘                  └─────────────────┘
```

## Requirements

- Kubernetes 1.19+
- Helm 3.0+
- Cluster with eBPF support for Beyla

## Uninstall

```bash
helm uninstall my-servicegraph
```
