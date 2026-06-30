---
name: sql-performance
version: 1.0.0
description: SQL query optimization and database schema review skill
---

## SQL Performance Review Instructions

When reviewing SQL queries and database schema, follow these guidelines:

1. Check for missing indexes on foreign keys and WHERE clause columns
2. Identify full table scans in queries with proper EXPLAIN ANALYZE
3. Review JOIN order for query plan efficiency
4. Avoid SELECT * in production queries; always specify columns
5. Use batch INSERT/UPDATE for bulk operations, not row-by-row
6. Check N+1 query patterns in ORM-generated queries
7. Verify connection pooling configuration matches workload
8. Review migration scripts for backward compatibility
9. Use appropriate lock modes to prevent deadlocks
10. Archive old data using partitioning or time-based retention policies
