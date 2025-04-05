# AsyncBatch

[中文](README.zh.md) | English

Generic asynchronous batch processor for Go, designed for efficient handling of large volumes of data through batch execution to reduce processing overhead. Supports configurable flushing strategies.

## Features

- **Automatic Batching**: Processes items when batch size or interval threshold is reached
- **Thread-Safe**: Concurrent submission and processing
- **Graceful Shutdown**: Ensures all pending items are processed
- **Generic Support**: Works with any data type via Go generics

## Use Cases

- Bulk database operations (inserts/updates)
- Batch API request aggregation
- Log/event processing pipelines
- Queue consumer implementations

## Installation
```sh
 go get github.com/kaichao/gopkg/asyncbatch
```

## Quick Start

```go
processor := asyncbatch.New[string](
    5,              // Batch size 
    10,             // Queue size
    2*time.Second, // Flush interval
    func(batch []string) {
        fmt.Println("Processing:", batch)
    },
)

// Submit items
for i := 0; i < 20; i++ {
    processor.Submit(fmt.Sprintf("item-%d", i))
}

// Shutdown cleanly
processor.Stop()
```

## API Reference

### `New`
Creates a new asynchronous batch processing queue.
```go
func New[T any](batchSize int, queueSize int, flushInterval time.Duration, processFunc func([]T)) *AsyncBatch[T]
```

### `Submit`
Submits a single element to the asynchronous queue.
```go
func (ab *AsyncBatch[T]) Submit(item T)
```

### `Stop`
Stops the asynchronous batch processing queue, ensuring all data is processed before exit.
```go
func (ab *AsyncBatch[T]) Stop()
```

