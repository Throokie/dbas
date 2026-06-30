---
name: data-pipeline
version: 1.0.0
description: Data engineering and ETL pipeline design, orchestration, and monitoring best practices
---

## Data Pipeline Instructions

When building data pipelines, follow these guidelines:

1. **Idempotency**: Every pipeline should produce the same result when re-run with the same input. Use upsert semantics and deduplication keys.
2. **Incremental Processing**: Prefer incremental loads over full refreshes. Use watermark columns (updated_at, event_time) to track processed data.
3. **Data Quality**: Implement quality checks at each stage: schema validation, null ratio checks, row count thresholds, referential integrity.
4. **Monitoring**: Alert on latency (time since last successful run), data volume anomalies, and error rates. Track SLAs per pipeline.
5. **Orchestration**: Use DAG-based orchestration (Airflow, Dagster, Prefect). Define task dependencies explicitly, avoid implicit ordering.
6. **Schema Evolution**: Use schema registry (Avro/Protobuf) with compatibility modes. Plan for backward and forward compatibility.
7. **Partitioning**: Partition large datasets by date/event_time. Use Hive-style partitioning for object storage, range partitioning for OLAP databases.
8. **Error Handling**: Implement dead letter queues for failed records. Never silently drop records — log and alert on all failures.
9. **Cost Control**: Set cost budgets per pipeline. Monitor storage vs compute tradeoffs. Use spot/preemptible instances for fault-tolerant workloads.
10. **Testing**: Unit test transforms, integration test with sample data, and run dry-run mode before production deployments.
11. **Documentation**: Maintain a data catalog with table lineage, owner, freshness expectations, and PII classification.
12. **Security**: Encrypt data at rest and in transit. Mask PII in non-production environments. Implement column-level access control for sensitive fields.
