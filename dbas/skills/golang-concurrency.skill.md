---
name: golang-concurrency
version: 1.0.0
description: Go concurrency patterns, goroutine management, and synchronization best practices
---

## Go Concurrency Instructions

When writing concurrent Go code, follow these guidelines:

1. **Channel Ownership**: Let the producer close the channel. Document who sends and who receives. Never close a channel from the receiver side.
2. **Goroutine Leaks**: Ensure every goroutine can be cancelled. Use context.Context as the first parameter for any function that spawns goroutines.
3. **Select Patterns**: Use select with default for non-blocking operations. Always handle context cancellation in select statements.
4. **Error Handling**: Use errgroup (golang.org/x/sync/errgroup) for fan-out/fan-in patterns. Propagate errors via dedicated error channels or errgroup.Wait().
5. **Fan-Out/Fan-In**: Distribute work across N goroutines, collect results. Use bounded worker pools to control concurrency. Cap at runtime.NumCPU() or configurable limit.
6. **Mutex vs Channels**: Use mutexes for protecting shared state, channels for communicating results. Prefer sync.Map for read-heavy concurrent maps.
7. **Context Propagation**: Always propagate context through the call chain. Use context.WithTimeout for external calls, context.WithCancel for graceful shutdown.
8. **Semaphore Pattern**: Use channel-based semaphores or weighted semaphore (x/sync/semaphore) to limit concurrent resource access.
9. **Once and Pool**: Use sync.Once for lazy singleton initialization. Use sync.Pool for frequently allocated temporary objects.
10. **Race Detection**: Always run tests with -race flag. Never ship code with known data races. Use atomic operations for simple counters.
11. **Pipeline Pattern**: Compose stages connected by channels. Each stage runs in its own goroutine. Handle stage cancellation and cleanup.
12. **Testing**: Test concurrent code with https://github.com/stretchr/testify/require.Eventually. Use goleak to detect goroutine leaks in tests.
