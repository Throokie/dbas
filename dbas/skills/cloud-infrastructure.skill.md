---
name: cloud-infrastructure
version: 1.0.0
description: Cloud infrastructure design, multi-cloud architecture, networking, and cost optimization
---

## Cloud Infrastructure Instructions

When designing cloud infrastructure, follow these guidelines:

1. **Well-Architected Framework**: Follow the five pillars: Operational Excellence, Security, Reliability, Performance Efficiency, Cost Optimization. Document tradeoffs per pillar.
2. **Multi-AZ / Multi-Region**: Deploy across at least 3 availability zones in production. Use active-passive for multi-region with Route53 failover or Traffic Director.
3. **Networking**: Use a hub-and-spoke VPC topology. Centralize egress through a NAT Gateway / Transit Gateway in the hub VPC. Implement VPC flow logs for audit.
4. **Identity & Access**: Use IAM roles (not users) for workloads. Implement least-privilege with policy conditions (source IP, MFA, time-bound). Use OIDC for CI/CD access.
5. **Container Orchestration**: Use managed Kubernetes (EKS, GKE, AKS) with node auto-repair and cluster autoscaler. Use pod security standards (restricted profile).
6. **Serverless**: Use serverless for event-driven, variable-load workloads. Set reserved concurrency to prevent runaway scaling. Use Cold Start optimization (Lambda SnapStart, provisioned concurrency).
7. **Database**: Choose purpose-built databases (Aurora for RDBMS, DynamoDB/Cassandra for NoSQL, ElastiCache for cache). Implement automated backups with PITR.
8. **Disaster Recovery**: Define RTO and RPO per workload tier. Implement backup → restore (RTO: hours), pilot light (RTO: minutes), or active-active (RTO: seconds).
9. **Auto Scaling**: Set min/max/desired for compute. Use target tracking scaling policies based on CPU, memory, or request count. Test scaling limits quarterly.
10. **Cost Optimization**: Use reserved instances / savings plans for steady-state. Use spot instances for fault-tolerant, stateless workloads. Set budget alerts at 50%, 80%, 100%.
11. **Compliance**: Implement audit logging (CloudTrail, Audit Logs). Encrypt all data at rest with KMS and in transit with TLS 1.3. Use Config rules for continuous compliance.
12. **Incident Response**: Define runbooks for common failure scenarios. Test game days quarterly. Use infrastructure automation for self-healing (auto-replace unhealthy instances).
