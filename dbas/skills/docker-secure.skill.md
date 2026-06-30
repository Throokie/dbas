---
name: docker-secure
version: 1.0.0
description: Docker security hardening and container best practices skill
---

## Docker Security Hardening Instructions

When building and reviewing Docker images, follow these practices:

1. Use distroless or scratch base images when possible
2. Never run containers as root; use USER directive with non-root user
3. Pin base image digests, not tags, for reproducible builds
4. Multi-stage builds to minimize final image size and attack surface
5. Scan images for CVEs with Trivy or Grype before deployment
6. Set read-only root filesystem with `--read-only` flag
7. Drop all capabilities with `--cap-drop=ALL` and add only needed ones
8. Use secrets mount instead of ENV for sensitive data at build time
9. Set resource limits (CPU/memory) for all containers
10. Add HEALTHCHECK instruction for proper container lifecycle management
