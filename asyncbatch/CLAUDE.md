# CLAUDE.md

## asyncbatch Package

Generic batch processor for asynchronous task processing with dynamic flow control and parallel execution.

### Core Type
```go
type BatchProcessor[T any] struct { ... }
func NewBatchProcessor[T any](handler func([]T), opts ...Option) (*BatchProcessor[T], error)
```

### Methods
- `Add(task T)` — Enqueue a task
- `Shutdown()` — Graceful shutdown, process remaining tasks

### Configuration Options
```go
asyncbatch.WithMaxSize(100)       // Max tasks per batch (default: 1000)
asyncbatch.WithLowerRatio(0.1)    // Min ratio for underfilled batches (default: 0.1)
asyncbatch.WithFixedWait(5*time.Millisecond)     // Initial wait (default: 5ms)
asyncbatch.WithUnderfilledWait(20*time.Millisecond) // Wait for underfilled (default: 20ms)
asyncbatch.WithNumWorkers(2)      // Parallel workers 1-8 (default: 1)
```

### Usage Example
```go
bp, _ := asyncbatch.NewBatchProcessor[int](
    func(batch []int) { fmt.Printf("Processing: %v\n", batch) },
    asyncbatch.WithMaxSize(100),
    asyncbatch.WithNumWorkers(2),
)
defer bp.Shutdown()

for i := 0; i < 500; i++ {
    bp.Add(i)
}
```
