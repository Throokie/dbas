---
name: observability
version: 1.0.0
description: System observability with logging, metrics, and tracing skill
---

## Observability Instructions

When implementing observability, follow these practices:

1. Use structured logging (JSON format) with consistent field names
2. Include correlation IDs in all logs for request tracing
3. Add RED metrics: Rate, Errors, Duration for every service endpoint
4. Export metrics in OpenTelemetry or Prometheus format
5. Implement distributed tracing with W3C trace context propagation
6. Set up health check endpoints (/health, /ready) for orchestration
7. Define SLOs for critical user journeys and monitor burn rate
8. Use log levels correctly: DEBUG for dev, INFO for normal, WARN for issues, ERROR for failures
9. Avoid logging sensitive data: PII, credentials, tokens, payment info
10. Create dashboards for: latency heatmaps, error rates, throughput, saturation
