---
name: devops-cicd
version: 1.0.0
description: CI/CD pipeline design, DevOps automation, and infrastructure-as-code best practices
---

## DevOps CI/CD Instructions

When designing CI/CD pipelines and DevOps workflows, follow these guidelines:

1. **Pipeline as Code**: Define all pipelines in version control (GitHub Actions, GitLab CI, Tekton). Never configure CI manually in a web UI.
2. **Stage Design**: Structure pipelines as: lint → unit test → build → integration test → security scan → deploy staging → E2E test → deploy production.
3. **Artifact Management**: Build once, deploy many. Store immutable artifacts (Docker images, binaries) with content-addressable tagging (git SHA).
4. **Environment Parity**: Keep dev/staging/prod as identical as possible. Use Docker Compose or Tilt for local development parity.
5. **Secrets Management**: Use a secrets vault (Vault, AWS Secrets Manager, SOPS) integrated with CI. Never store secrets in pipeline YAML or environment variables.
6. **Deployment Strategies**: Use blue-green, canary, or rolling deployments. Implement automated rollback on health check failure within the deployment window.
7. **Infrastructure as Code**: Manage all infrastructure with Terraform, Pulumi, or CloudFormation. Use state locking and remote state backends.
8. **Monitoring & Alerting**: Deploy alongside infrastructure: metrics dashboard, centralized logging (Loki/ELK), and alerting rules (Prometheus/Alertmanager).
9. **Security in Pipeline**: Run SAST (Semgrep/SonarQube) on every commit. Run DAST on staging. Scan container images with Trivy or Grype. Enforce SLSA levels.
10. **Cost Management**: Tag all cloud resources with cost center, environment, and project. Set budget alerts. Schedule non-production resources to shut down overnight.
11. **Backup & Disaster Recovery**: Automate database backups with point-in-time recovery. Document and test DR runbook quarterly.
12. **SLOs and Error Budgets**: Define SLOs for service uptime and latency. Use error budgets to balance velocity and reliability. Gate deployments when budget is exhausted.
