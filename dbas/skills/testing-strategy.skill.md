---
name: testing-strategy
version: 1.0.0
description: Software testing strategy and test-driven development skill
---

## Testing Strategy Instructions

When planning and implementing tests, follow these guidelines:

1. Follow the testing trophy: static analysis > unit tests > integration > E2E
2. Write tests before code (TDD) for core business logic
3. Use table-driven tests for multiple input/output scenarios
4. Mock external dependencies (network, database, filesystem) at boundaries
5. Aim for 80%+ coverage on business logic, 50%+ overall
6. Run tests in CI: unit tests on every push, integration nightly
7. Use property-based testing for data transformation functions
8. Test error paths and edge cases, not just happy paths
9. Keep tests independent: no shared state, no test ordering dependencies
10. Write test descriptions that document the expected behavior
