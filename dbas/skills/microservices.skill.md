---
name: microservices
version: 1.0.0
description: Microservices architecture patterns, service decomposition, and inter-service communication best practices
---

## Microservices Architecture Instructions

When designing microservices architectures, follow these guidelines:

1. **Service Boundaries**: Decompose by business capability, not technical layer. Each service owns its data and domain logic.
2. **Inter-Service Communication**: Prefer asynchronous messaging (Kafka/RabbitMQ) over synchronous HTTP for cross-service workflows. Use gRPC for low-latency synchronous calls.
3. **API Gateway**: Route all external traffic through an API gateway (Kong, Envoy, or custom). Handle auth, rate limiting, and routing at the gateway layer.
4. **Service Discovery**: Use DNS-based or registry-based discovery (Consul, Eureka, Kubernetes DNS). Avoid hardcoded service URLs.
5. **Circuit Breaker**: Implement circuit breaker pattern (Hystrix, Resilience4j, or Istio) for fault isolation. Fail fast, degrade gracefully.
6. **Distributed Tracing**: Instrument all services with OpenTelemetry. Propagate trace context via W3C TraceContext headers.
7. **Saga Pattern**: For distributed transactions, use choreography-based or orchestration-based sagas. Avoid two-phase commit (2PC) across services.
8. **Observability**: Each service must expose health (`/healthz`), metrics (`/metrics`), and readiness (`/readyz`) endpoints.
9. **Database per Service**: Each microservice owns its database schema. Never share databases across services. Use event-driven data synchronization.
10. **Deployment**: Containerize each service independently. Use Kubernetes Deployments with resource limits, liveness, and readiness probes.
11. **Versioning**: Version APIs from day one. Maintain backward compatibility for at least one major version.
12. **Testing**: Write contract tests (Pact) between services. Use consumer-driven contracts to detect breaking changes early.
