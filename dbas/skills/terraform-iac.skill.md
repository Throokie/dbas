---
name: terraform-iac
version: 1.0.0
description: Terraform infrastructure-as-code development and review skill
---

## Terraform Development Instructions

When writing or reviewing Terraform code, follow these practices:

1. Use remote state backends (S3/Blob/GCS) with state locking enabled
2. Organize modules by responsibility: networking, compute, data, security
3. Pin provider versions using required_providers block
4. Use variables with type constraints and validation blocks
5. Output only necessary values; never output secrets
6. Tag all resources with consistent project/environment/owner tags
7. Use workspaces or terragrunt for environment separation
8. Implement lifecycle policies to prevent accidental destruction
9. Run plan before apply in CI pipelines
10. Use .terraformignore to exclude unnecessary files from uploads
