---
name: api-design
version: 1.0.0
description: REST API design and documentation best practices skill
---

## REST API Design Instructions

When designing REST APIs, follow these guidelines:

1. Use nouns for resources, verbs only for actions (POST /orders, not /createOrder)
2. Version APIs via URL prefix (/v1/, /v2/) or content negotiation
3. Use standard HTTP status codes consistently (201 Created, 400 Bad Request, etc.)
4. Implement pagination for list endpoints with cursor or offset/limit
5. Use snake_case or camelCase consistently across all endpoints
6. Validate all inputs with schema validation (JSON Schema, OpenAPI)
7. Return consistent error format: {error: {code, message, details}}
8. Rate limit APIs with meaningful headers (X-RateLimit-Remaining)
9. Document all endpoints with OpenAPI/Swagger specification
10. Implement idempotency keys for mutating operations
