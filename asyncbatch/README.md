# AsyncBatch

English | [中文](README.zh.md)

`asyncbatch` is a Go package for asynchronous batch processing, built with generics, allowing users to configure batch size, wait times, and processing logic. It is suitable for high-throughput or low-latency scenarios.

## Features

- **Generic Support**: Supports tasks of any type with type safety.
- **Flexible Configuration**: Configure batch processing parameters using `With...` functions.
- **Dynamic Wait Times**: Adjusts wait times based on batch state (empty, partial, full).
- **Graceful Shutdown**: Safely exits after processing remaining tasks.

## Use Cases

- Batch database operations (insert/update)
- Aggregating API requests
- Log/event processing pipelines
- Queue consumer implementations

## Installation

```sh
go get github.com/kaichao/gopkg/asyncbatch
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    "github.com/kaichao/gopkg/asyncbatch"
)

func main() {
    // Create a batch processor
    bp := asyncbatch.NewBatchProcessor[string](
        asyncbatch.WithMaxSize[string](3),
        asyncbatch.WithMaxWait[string](time.Second),
        asyncbatch.WithEmptyWait[string](200*time.Millisecond),
        asyncbatch.WithPartialWait[string](100*time.Millisecond),
        asyncbatch.WithWorker[string](func(batch []string) {
            fmt.Printf("Processing batch: %v\n", batch)
        }),
    )

    // Start batch processing
    go bp.Run()

    // Add tasks
    bp.Add("task1")
    bp.Add("task2")
    bp.Add("task3")

    // Wait for processing
    time.Sleep(200 * time.Millisecond)

    // Shutdown
    bp.Shutdown()
}
```

## API
### Types
- `BatchProcessor[T any]`: A batch processor using generics, supporting tasks of any type.
- `Option[T any]`: A configuration function type for setting parameters.

### Functions
- **`NewBatchProcessor[T any](opts ...Option[T]) *BatchProcessor[T]`**  
  Creates a batch processor with the provided configuration options. Defaults: max batch size 100, wait times 1 second.
- **`WithMaxSize[T any](size int) Option[T]`**  
  Sets the maximum batch size (effective if >0).
- **`WithMaxWait[T any](duration time.Duration) Option[T]`**  
  Sets the maximum wait time for a full batch (effective if >0). Full batches are typically processed immediately; `maxWait` limits the maximum delay for consecutive full batches.
- **`WithEmptyWait[T any](duration time.Duration) Option[T]`**  
  Sets the wait time for an empty batch (effective if >0). Empty batches wait for new tasks.
- **`WithPartialWait[T any](duration time.Duration) Option[T]`**  
  Sets the wait time for a partial batch (effective if >0). Partial batches wait for more tasks or process on timeout.
- **`WithWorker[T any](worker func([]T)) Option[T]`**  
  Sets the batch processing function to handle a batch of tasks.
- **`(*BatchProcessor[T]) Add(task T) error`**  
  Adds a task to the batch processor. Returns an error if the processor is closed or the task channel is full.
- **`(*BatchProcessor[T]) Run()`**  
  Starts the batch processor to process tasks asynchronously.
- **`(*BatchProcessor[T]) Shutdown()`**  
  Shuts down the batch processor, processing remaining tasks before exiting.
- **`(*BatchProcessor[T]) MaxSize() int`**  
  Returns the maximum batch size.
- **`(*BatchProcessor[T]) MaxWait() time.Duration`**  
  Returns the maximum wait time for a full batch.
- **`(*BatchProcessor[T]) EmptyWait() time.Duration`**  
  Returns the wait time for an empty batch.
- **`(*BatchProcessor[T]) PartialWait() time.Duration`**  
  Returns the wait time for a partial batch.
- **`(*BatchProcessor[T]) Worker() func([]T)`**  
  Returns the batch processing function.

## Wait Time Mechanism
`asyncbatch` dynamically controls batch processing triggers using three wait times to optimize throughput and latency:

- **`maxWait`**: The maximum wait time for a full batch (task count ≥ `maxSize`).  
  - **Purpose**: Full batches are processed immediately; `maxWait` limits the maximum delay for consecutive full batches in high-throughput scenarios to prevent task accumulation.  
  - **Example**: With `maxSize=3`, `maxWait=1s`, three tasks are processed immediately. If tasks arrive continuously, processing occurs at most every 1 second.
- **`emptyWait`**: The wait time for an empty batch (no tasks).  
  - **Purpose**: Waits for new tasks during idle periods, reducing resource usage.  
  - **Example**: With `emptyWait=200ms`, an empty batch checks for new tasks every 200ms.
- **`partialWait`**: The wait time for a partial batch (task count < `maxSize`).  
  - **Purpose**: Waits for more tasks or processes on timeout, ideal for low-throughput scenarios.  
  - **Example**: With `partialWait=100ms`, two tasks wait 100ms before processing, resetting the wait if new tasks arrive.

**Mechanism**: The `Run()` method selects the appropriate wait time based on the batch state (empty, partial, full), dynamically adjusting the timer. Full batches are prioritized, partial batches balance latency and throughput, and empty batches conserve resources. On shutdown, all remaining tasks are processed.
