---
name: database-design
version: 1.0.0
description: Database schema design, indexing strategies, query optimization, and data modeling principles
---

## Database Design Instructions

When designing database schemas, follow these guidelines:

1. **Normalization**: Normalize to 3NF by default. Denormalize only for read-heavy query patterns after measuring actual performance. Document denormalization decisions.
2. **Naming Conventions**: Use snake_case for table and column names. Use singular nouns for tables (user, not users). Prefix join tables with two table names (user_role).
3. **Primary Keys**: Use UUIDv7 (time-sortable) for distributed systems, auto-increment BIGINT for single-server OLTP. Never use natural keys as PKs.
4. **Indexing**: Index columns used in WHERE, JOIN, ORDER BY. Use composite indexes with column order matching query patterns. Avoid over-indexing write-heavy tables.
5. **Query Optimization**: Use EXPLAIN ANALYZE before writing any query for production. Target index-only scans or index seeks. Avoid table scans on large tables.
6. **Foreign Keys**: Enforce referential integrity with foreign key constraints. Define ON DELETE (CASCADE, SET NULL, or RESTRICT) explicitly for each relationship.
7. **Migrations**: Use versioned, reversible migrations (golang-migrate, Flyway). Test rollback before deploying forward. Never modify past migration files.
8. **Partitioning**: Partition tables over 100GB by date (range) or tenant ID (list). Ensure queries include the partition key to enable partition pruning.
9. **Connection Pooling**: Use connection pools with max open connections, max idle connections, and max lifetime. Set pool size based on database max_connections and workload.
10. **Transactions**: Keep transactions short. Never hold transactions across network calls. Use pessimistic locking (SELECT FOR UPDATE) sparingly.
11. **Read Replicas**: Offload reporting and analytics queries to read replicas. Use read-write splitting at the application or proxy (PgBouncer, ProxySQL) layer.
12. **Data Archival**: Define a data retention policy. Archive old data to colder storage (S3 Glacier, Google Coldline) using partitioned table detach or time-based delete.
