---
name: rust-embedded
version: 1.0.0
description: Rust embedded systems development, no_std patterns, and hardware abstraction best practices
---

## Rust Embedded Development Instructions

When developing embedded systems with Rust, follow these guidelines:

1. **no_std First**: Start with `#![no_std]` and #[entry] from the cortex-m-rt or riscv-rt crate. Only use alloc when absolutely necessary.
2. **Peripheral Access**: Use PAC (Peripheral Access Crate) for register-level access. Build HAL (Hardware Abstraction Layer) on top of PAC for reusable drivers.
3. **Type-State Pattern**: Use Rust's type system to enforce hardware configuration at compile time. Prevent invalid states (e.g., unconfigured GPIO) with zero-cost abstractions.
4. **Interrupt Handling**: Use cortex-m-rt #[interrupt] attribute. Keep ISRs short — set flags, don't do heavy processing. Use critical sections sparingly.
5. **RTOS vs Bare Metal**: Prefer RTIC (Real-Time Interrupt-driven Concurrency) for complex scheduling. Use embassy for async embedded with multi-task support.
6. **Memory Constraints**: Measure stack usage with cargo-call-stack. Use fixed-size buffers (heapless::Vec, arrayvec) instead of heap allocation.
7. **Debugging**: Use probe-rs + VS Code for debugging. Set up RTT (Real-Time Transfer) for logging without blocking execution.
8. **Testing**: Run unit tests on host with mocked peripherals. Use defmt-test for on-target testing. Run integration tests with probe-rs run.
9. **Firmware Updates**: Implement OTA or DFU (Device Firmware Update) from day one. Partition flash into bootloader + application + fallback slots.
10. **Power Management**: Use WFI (Wait For Interrupt) in idle loops. Gate clocks to unused peripherals. Implement multiple sleep modes based on wake latency requirements.
11. **Safety-Critical**: For safety-critical applications, use Ferrocene language specification and follow IEC 62304 or ISO 26262 guidelines.
12. **Toolchain**: Use Rustup targets for your architecture (thumbv7em-none-eabihf, riscv32imac-unknown-none-elf). Use cargo-binutils for size analysis.
