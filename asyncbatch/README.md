# AsyncBatch

English | [中文](README.zh.md)

A general-purpose asynchronous batch processing tool in Go, supporting configurable flush strategies and dynamic wait mechanisms.

## Features

- **Automatic Batching**: Automatically processes batches when the batch size or time interval is reached.
- **Dynamic Partial Wait**: Uses exponentially increasing wait times (`partialWait * (1 << partialCount)`) to accumulate more items when the batch is not full, reducing batch fragmentation.
- **Generics Support**: Supports any data type via Go generics.
- **Thread-Safe**: Supports concurrent submission and processing.
- **Graceful Shutdown**: Ensures all pending items are processed.
- **Multi-Processing Backends**: Speeds up computation response times.

## Typical Use Cases

- Batch database operations (insert/update)
- API request aggregation
- Log/event processing pipelines
- Queue consumer implementations

## Installation

```sh
go get github.com/kaichao/gopkg/asyncbatch
```

## Quick Start

```go
// Define the processing function
processFunc := func(items []string) error {
    fmt.Printf("Processing batch: %v", items)
    return nil
}

// Create a Batch with 3 workers, batch size of 5, and 2-second flush interval
batch := asyncbatch.New[string](
    processFunc,
    asyncbatch.WithNumWorkers(3),
    asyncbatch.WithBatchSize(5),
    asyncbatch.WithFlushInterval(2*time.Second),
)

// Enqueue tasks
for i := 0; i < 12; i++ {
    batch.Push(fmt.Sprintf("item-%d", i))
}

// Wait for a period to trigger flushing
time.Sleep(5 * time.Second)

// Close to process remaining tasks and stop workers
batch.Close()
```

### API Description

```go
type Batch[T any] struct {
    // Push(item) adds a task to the queue
    Push(item T)

    // Close() closes the queue, processes remaining tasks, and stops all workers
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

- **`processFunc`**  
  Function to process a batch of tasks, receives `[]T` and returns an error. Errors are ignored internally.

- **`opts`**  
  Optional configuration options (see below).

#### Optional Configuration

- **`WithNumWorkers(n int)`**  
  Number of concurrent workers. Default: `1`. Minimum: `1`.

- **`WithBatchSize(size int)`**  
  Maximum number of tasks per batch. Default: `1`. Minimum: `1`.

- **`WithFlushInterval(d time.Duration)`**  
  Maximum wait time before flushing non-empty batches. Default: `0` (disables timed flushing, only triggers on full batch or Close).

#### Default Parameters

- **Channel Capacity**: `100`
- **Number of Workers**: `1`
- **Batch Size**: `1`
- **Flush Interval**: `0` (only processes on full batch or Close)

## Logging

Workers output logs for startup, batch processing, timed flushing, and shutdown:

```
worker-12345 started
worker-12345 processing 5 items
worker-12345 flushing 2 items
worker-12345 stopped
```

Logs use the standard `log.Printf`.

## New Features

Below is a detailed description of the new features:

- **Dynamic Partial Wait Mechanism**:  
  When a batch does not reach `batchSize`, the system waits with exponentially increasing times (e.g., 50ms, 100ms, 200ms) to allow more tasks to join, reducing unnecessary partial batch processing. This mechanism is particularly suited for scenarios with uneven task arrival rates.  
  **Example**: In the test `TestPartialBatchWait`, after sending 5 items, the system dynamically waits to accumulate 8 items before processing them as a single batch.

- **Improved Timer Logic**:  
  When the timer triggers, the system attempts to read more data from the queue until `batchSize` is reached or the queue is empty. This improvement ensures batches are as complete as possible, enhancing processing efficiency.  
  **Example**: In the test `TestDynamicPartialWait`, after sending 5 items, the system dynamically extends the wait time to accumulate 10 items before processing.

## Test Verification

You can verify the new features with the following commands:

```bash
go test -timeout 30s -run ^TestPartialBatchWait$ github.com/kaichao/gopkg/asyncbatch
go test -timeout 30s -run ^TestDynamicPartialWait$ github.com/kaichao/gopkg/asyncbatch
```

- **Expected Results**:  
  - `TestPartialBatchWait`: Processes 8 items in 1 batch.  
  - `TestDynamicPartialWait`: Processes 10 items in 1 batch.