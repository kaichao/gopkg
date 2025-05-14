# AsyncBatch

[中文](README.zh.md) | English

Generic asynchronous batch processor for Go, designed for efficient handling of large volumes of data through batch execution to reduce processing overhead. Supports configurable flushing strategies.

## Features

- **Automatic Batching**: Processes items when batch size or interval threshold is reached
- **Generic Support**: Works with any data type via Go generics
- **Thread-Safe**: Concurrent submission and processing
- **Graceful Shutdown**: Ensures all pending items are processed
- **Multiprocess Backend**: Accelerates computation response speed

## Use Cases

- Bulk database operations (inserts/updates)
- Batch API request aggregation
- Log/event processing pipelines
- Queue consumer implementations

## Installation

```bash
go get github.com/kaichao/gopkg/asyncbatch
```

## Usage

```go
    // Define a processing function
    processFunc := func(items []string) error {
        fmt.Printf("Processing batch: %v", items)
        return nil
    }

    // Create a Batch with 3 workers, batch size 5, flush every 2s
    batch := asyncbatch.New[string](
        processFunc,
        asyncbatch.WithNumWorkers(3),
        asyncbatch.WithBatchSize(5),
        asyncbatch.WithFlushInterval(2*time.Second),
    )

    // Push items
    for i := 0; i < 12; i++ {
        batch.Push(fmt.Sprintf("item-%d", i))
    }

    // Give time for flushes
    time.Sleep(5 * time.Second)

    // Close to finish any remaining items and stop workers
    batch.Close()
```

### API

```go
// Batch[T] is the main type.
type Batch[T any] struct {
    // Push(item) submits an item to the batch queue.
    Push(item T)

    // Close() stops accepting new items, flushes pending items, and stops all workers.
    Close()
}
```

#### Constructor

```go
func New[T any](
    processFunc func([]T) error,
    opts ...Option,
) *Batch[T]
```

- `processFunc`  
  Function called on each batch of `[]T`. Should return an error if processing failed (errors are currently ignored inside the worker).

- `opts`
  Functional options to customize behavior (see below).

#### Options

- `WithNumWorkers(n int)`
  Number of concurrent worker goroutines. Default: `1`. Minimum: `1`.

- `WithBatchSize(size int)`  
  Number of items per batch before immediate processing. Default: `1`. Minimum: `1`.

- `WithFlushInterval(d time.Duration)`
  Maximum time to wait before flushing a non‑empty batch. Default: `0` (no periodic flush unless set).

#### Default Settings

- **Channel (queue) size**: `100`
- **Number of workers**: `1`
- **Batch size**: `1`
- **Flush interval**: `0` (disabled; batch only flushes when full or on close)

## Logging

Workers log start/stop and each batch/flush:

```
worker-12345 started
worker-12345 processing 5 items
worker-12345 flushing 2 items
worker-12345 stopped
```

Logs go to the standard logger (`log.Printf`).

