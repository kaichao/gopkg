# asyncbatch

`asyncbatch` is a Go package that provides a generic batch processor for asynchronous task processing. It collects tasks into batches and processes them according to configurable rules, optimizing throughput and resource usage.

## Features

- **Asynchronous Processing**: Automatically starts processing tasks upon creation.
- **Batch Size Control**: Configurable maximum batch size (`maxSize`).
- **Continuous Processing**: Processes non-empty batches immediately if the previous batch size meets the upper threshold (`upperRatio`).
- **Flexible Waiting**:
  - Waits `fixedWait` to check for tasks.
  - Processes batches with at least `lowerRatio * maxSize` tasks after `fixedWait`.
  - Waits `underfilledWait` for underfilled batches before processing.
- **Parallel Workers**: Supports 1 to 8 concurrent workers (`numWorkers`).
- **Graceful Shutdown**: Processes all remaining tasks before stopping.

## Usage

### Installation

```bash
go get github.com/kaichao/gopkg/asyncbatch
```

### Example

```go
package main

import (
    "fmt"
    "github.com/kaichao/gopkg/asyncbatch"
    "time"
)

func main() {
    // Create a batch processor
    bp, err := asyncbatch.NewBatchProcessor[string](
        asyncbatch.WithMaxSize[string](10),
        asyncbatch.WithUpperRatio[string](0.5),
        asyncbatch.WithLowerRatio[string](0.1),
        asyncbatch.WithFixedWait[string](100*time.Millisecond),
        asyncbatch.WithUnderfilledWait[string](200*time.Millisecond),
        asyncbatch.WithNumWorkers[string](2),
        asyncbatch.WithWorker[string](func(batch []string) {
            fmt.Printf("Processing batch of %d tasks: %v\n", len(batch), batch)
        }),
    )
    if err != nil {
        panic(err)
    }
    defer bp.Shutdown()

    // Add tasks
    for i := 0; i < 25; i++ {
        bp.Add(fmt.Sprintf("task%d", i))
    }

    // Wait briefly to observe processing
    time.Sleep(time.Second)
}
```

### Configuration Options

- `WithMaxSize(size int)`: Sets the maximum batch size (default: 100).
- `WithUpperRatio(ratio float64)`: Sets the upper ratio for continuous processing (default: 0.5).
- `WithLowerRatio(ratio float64)`: Sets the lower ratio for underfilled waiting (default: 0.1).
- `WithFixedWait(duration time.Duration)`: Sets the initial wait time (default: 100ms).
- `WithUnderfilledWait(duration time.Duration)`: Sets the wait time for underfilled batches (default: 500ms).
- `WithNumWorkers(numWorkers int)`: Sets the number of parallel workers (1 to 8, default: 1).
- `WithWorker(worker func([]T))`: Sets the batch processing function (required).

### Constraints

- `0 < upperRatio, lowerRatio <= 1`, and `upperRatio >= lowerRatio`.
- `fixedWait < underfilledWait`.
- `numWorkers` must be between 1 and 8.
- `maxSize` must be positive.
- Worker function must be provided.

## Testing

Run unit tests:

```bash
go test -v ./...
```

Check code coverage:

```bash
go test -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

