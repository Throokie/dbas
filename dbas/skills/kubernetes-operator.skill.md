---
name: kubernetes-operator
version: 1.0.0
description: Kubernetes operator development and code review skill
---

## Kubernetes Operator Development Instructions

When developing or reviewing Kubernetes operators, follow these practices:

1. Use controller-runtime and kubebuilder patterns, not raw client-go
2. Implement proper leader election for high-availability deployments
3. Add finalizers for resource cleanup on deletion
4. Use predicates to filter unnecessary reconciliation events
5. Set OwnerReferences for garbage collection of child resources
6. Implement proper status conditions with LastTransitionTime
7. Use rate limiting and exponential backoff in reconcilers
8. Write envtest-based integration tests for controller logic
9. Ensure RBAC annotations match actual API operations
10. Handle webhook validation and conversion for version upgrades
