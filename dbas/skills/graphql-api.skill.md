---
name: graphql-api
version: 1.0.0
description: GraphQL API design, schema architecture, resolver patterns, and performance optimization
---

## GraphQL API Instructions

When designing GraphQL APIs, follow these guidelines:

1. **Schema-First Design**: Define the schema before implementing resolvers. Use SDL (Schema Definition Language) as the source of truth. Codegen types from schema.
2. **Naming Conventions**: Use camelCase for fields and arguments. Use PascalCase for types and enums. Prefix mutation names with verbs (createUser, updateOrder).
3. **DataLoader Pattern**: Use DataLoader (or Dataloaden for Go) to batch and cache database queries. Resolve N+1 queries for every list field with hasMany relationships.
4. **Pagination**: Implement cursor-based pagination (Relay Connections spec) for list fields. Never use offset-based pagination in GraphQL.
5. **Error Handling**: Use a discriminated union type for errors. Never throw exceptions in resolvers — return user-visible errors in the data response.
6. **Authorization**: Enforce auth at the field resolver level. Use a middleware that injects the authenticated user context. Implement field-level permission checks.
7. **Input Validation**: Define input types with constraints (non-null, regex, length limits). Validate all inputs in the resolver or a validation middleware layer.
8. **Complexity Limits**: Set query complexity limits and depth limits. Analyze incoming queries before execution. Reject abusive queries with a clear error.
9. **Subscriptions**: Use subscriptions for real-time events (not for queries). Over websocket with graphql-ws protocol. Handle reconnection and backpressure.
10. **Caching**: Implement persisted queries for production. Use CDN caching for public queries. Set Cache-Control hints on response extensions for field-level caching.
11. **Federation**: For microservice GraphQL, use Apollo Federation or GraphQL Mesh. Define entities with @key directive for cross-service references.
12. **Testing**: Write integration tests that execute real queries against a test server. Use snapshot testing for complex response shapes. Test error paths systematically.
