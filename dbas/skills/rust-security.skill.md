---
name: rust-security
version: 1.0.0
description: Rust security audit and unsafe code review skill
---

## Rust Security Audit Instructions

When auditing Rust code for security issues, follow these guidelines:

1. Review all `unsafe` blocks for pointer validity, alignment, and memory leaks
2. Check for proper use of `Send` and `Sync` trait implementations
3. Verify that `Pin` is used correctly for self-referential types
4. Audit FFI calls for correct C ABI compatibility and null pointer handling
5. Check for potential integer overflow/underflow in arithmetic operations
6. Verify error handling doesn't leak sensitive information via `Display`/`Debug`
7. Review dependency tree for outdated crates with known vulnerabilities
8. Ensure proper use of locking primitives (Mutex, RwLock) to prevent deadlocks
